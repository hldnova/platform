// Package executor contains implementations of backend.Executor
// that depend on the query service.
package executor

import (
	"context"
	"sync"
	"time"

	"github.com/influxdata/influxdb/logger"
	"github.com/influxdata/platform/query"
	"github.com/influxdata/platform/task/backend"
	"go.uber.org/zap"
)

// queryServiceExecutor is an implementation of backend.Executor that depends on a QueryService.
type queryServiceExecutor struct {
	svc    query.QueryService
	st     backend.Store
	logger *zap.Logger
}

var _ backend.Executor = (*queryServiceExecutor)(nil)

// NewQueryServiceExecutor returns a new executor based on the given QueryService.
// In general, you should prefer NewAsyncQueryServiceExecutor, as that code is smaller and simpler,
// because asynchronous queries are more in line with the Executor interface.
func NewQueryServiceExecutor(logger *zap.Logger, svc query.QueryService, st backend.Store) backend.Executor {
	return &queryServiceExecutor{logger: logger, svc: svc, st: st}
}

func (e *queryServiceExecutor) Execute(ctx context.Context, run backend.QueuedRun) (backend.RunPromise, error) {
	t, err := e.st.FindTaskByID(ctx, run.TaskID)
	if err != nil {
		return nil, err
	}

	return newSyncRunPromise(ctx, run, e, t), nil
}

// syncRunPromise implements backend.RunPromise for a synchronous QueryService.
type syncRunPromise struct {
	qr     backend.QueuedRun
	svc    query.QueryService
	t      *backend.StoreTask
	ctx    context.Context
	cancel context.CancelFunc
	logger *zap.Logger
	logEnd func()

	finishOnce sync.Once     // Ensure we set the values only once.
	ready      chan struct{} // Closed inside finish. Indicates Wait will no longer block.
	res        *runResult
	err        error
}

var _ backend.RunPromise = (*syncRunPromise)(nil)

func newSyncRunPromise(ctx context.Context, qr backend.QueuedRun, e *queryServiceExecutor, t *backend.StoreTask) *syncRunPromise {
	ctx, cancel := context.WithCancel(ctx)
	log, logEnd := logger.NewOperation(e.logger, "Executing task", "execute")
	rp := &syncRunPromise{
		qr:     qr,
		svc:    e.svc,
		t:      t,
		logger: log,
		logEnd: logEnd,
		ctx:    ctx,
		cancel: cancel,
		ready:  make(chan struct{}),
	}

	go rp.doQuery()
	go rp.cancelOnContextDone()

	return rp
}

func (p *syncRunPromise) Run() backend.QueuedRun {
	return p.qr
}

func (p *syncRunPromise) Wait() (backend.RunResult, error) {
	<-p.ready

	// Need an explicit return nil to avoid the non-nil interface value issue.
	if p.err != nil {
		return nil, p.err
	}
	return p.res, nil
}

func (p *syncRunPromise) Cancel() {
	p.finish(nil, backend.ErrRunCanceled)
}

func (p *syncRunPromise) finish(res *runResult, err error) {
	p.finishOnce.Do(func() {
		defer p.logEnd()

		// Always cancel p's context.
		// If finish is called before p.svc.Query completes, the query will be interrupted.
		// If afterwards, then p.cancel is just a resource cleanup.
		defer p.cancel()

		p.res, p.err = res, err
		close(p.ready)

		if err != nil {
			p.logger.Info("Execution failed to get result", zap.Error(err))
		} else if res.err != nil {
			p.logger.Info("Got result with error", zap.Error(res.err))
		} else {
			p.logger.Info("Completed successfully")
		}
	})
}

func (p *syncRunPromise) doQuery() {
	spec, err := query.Compile(p.ctx, p.t.Script, time.Unix(p.qr.Now, 0))
	if err != nil {
		p.finish(nil, err)
		return
	}

	req := &query.Request{
		OrganizationID: p.t.Org,
		Compiler: query.SpecCompiler{
			Spec: spec,
		},
	}
	it, err := p.svc.Query(p.ctx, req)
	if err != nil {
		// Assume the error should not be part of the runResult.
		p.finish(nil, err)
		return
	}

	// Drain the result iterator.
	for it.More() {
		// Is it okay to assume it.Err will be set if the query context is canceled?
		_ = it.Next()
	}

	p.finish(&runResult{err: it.Err()}, nil)
}

func (p *syncRunPromise) cancelOnContextDone() {
	select {
	case <-p.ready:
		// Nothing to do.
	case <-p.ctx.Done():
		// Maybe the parent context was canceled,
		// or maybe finish was called already.
		// If it's the latter, this call to finish will be a no-op.
		p.finish(nil, p.ctx.Err())
	}
}

// asyncQueryServiceExecutor is an implementation of backend.Executor that depends on an AsyncQueryService.
type asyncQueryServiceExecutor struct {
	svc    query.AsyncQueryService
	st     backend.Store
	logger *zap.Logger
}

var _ backend.Executor = (*asyncQueryServiceExecutor)(nil)

// NewQueryServiceExecutor returns a new executor based on the given AsyncQueryService.
func NewAsyncQueryServiceExecutor(logger *zap.Logger, svc query.AsyncQueryService, st backend.Store) backend.Executor {
	return &asyncQueryServiceExecutor{logger: logger, svc: svc, st: st}
}

func (e *asyncQueryServiceExecutor) Execute(ctx context.Context, run backend.QueuedRun) (backend.RunPromise, error) {
	t, err := e.st.FindTaskByID(ctx, run.TaskID)
	if err != nil {
		return nil, err
	}

	spec, err := query.Compile(ctx, t.Script, time.Unix(run.Now, 0))
	if err != nil {
		return nil, err
	}

	req := &query.Request{
		OrganizationID: t.Org,
		Compiler: query.SpecCompiler{
			Spec: spec,
		},
	}
	q, err := e.svc.Query(ctx, req)
	if err != nil {
		return nil, err
	}

	return newAsyncRunPromise(run, q, e), nil
}

// asyncRunPromise implements backend.RunPromise for an AsyncQueryService.
type asyncRunPromise struct {
	qr backend.QueuedRun
	q  query.Query

	logger *zap.Logger
	logEnd func()

	finishOnce sync.Once     // Ensure we set the values only once.
	ready      chan struct{} // Closed inside finish. Indicates Wait will no longer block.
	res        *runResult
	err        error
}

var _ backend.RunPromise = (*asyncRunPromise)(nil)

func newAsyncRunPromise(qr backend.QueuedRun, q query.Query, e *asyncQueryServiceExecutor) *asyncRunPromise {
	log, logEnd := logger.NewOperation(e.logger, "Executing task", "execute")

	p := &asyncRunPromise{
		qr:    qr,
		q:     q,
		ready: make(chan struct{}),

		logger: log,
		logEnd: logEnd,
	}

	go p.followQuery()
	return p
}

func (p *asyncRunPromise) Run() backend.QueuedRun {
	return p.qr
}

func (p *asyncRunPromise) Wait() (backend.RunResult, error) {
	<-p.ready

	// Need an explicit return nil to avoid the non-nil interface value issue.
	if p.err != nil {
		return nil, p.err
	}
	return p.res, nil
}

func (p *asyncRunPromise) Cancel() {
	p.finish(nil, backend.ErrRunCanceled)
}

// followQuery waits for the query to become ready and sets p's results.
// If the promise is finished somewhere else first, such as if it is canceled,
// followQuery will return.
func (p *asyncRunPromise) followQuery() {
	// Always need to call Done after query is finished.
	defer p.q.Done()

	select {
	case <-p.ready:
		// The promise was finished somewhere else, so we don't need to call p.finish.
		// But we do need to cancel the query. This could be a no-op.
		p.q.Cancel()
	case _, ok := <-p.q.Ready():
		if !ok {
			// Something went wrong with the query. Set the error in the run result.
			rr := &runResult{err: p.q.Err()}
			p.finish(rr, nil)
			return
		}

		// Otherwise, query was successful.
		// TODO(mr): collect query statistics, once RunResult interface supports them?
		p.finish(new(runResult), nil)
	}
}

func (p *asyncRunPromise) finish(res *runResult, err error) {
	p.finishOnce.Do(func() {
		defer p.logEnd()

		p.res, p.err = res, err
		close(p.ready)

		if err != nil {
			p.logger.Info("Execution failed to get result", zap.Error(err))
		} else if res.err != nil {
			p.logger.Info("Got result with error", zap.Error(res.err))
		} else {
			p.logger.Info("Completed successfully")
		}
	})
}

type runResult struct {
	err       error
	retryable bool
}

var _ backend.RunResult = (*runResult)(nil)

func (rr *runResult) Err() error        { return rr.err }
func (rr *runResult) IsRetryable() bool { return rr.retryable }
