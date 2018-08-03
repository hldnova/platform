package executor_test

import (
	"context"
	"errors"
	"math"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/influxdata/platform"
	"github.com/influxdata/platform/query"
	_ "github.com/influxdata/platform/query/builtin"
	"github.com/influxdata/platform/query/execute"
	"github.com/influxdata/platform/query/values"
	"github.com/influxdata/platform/task/backend"
	"github.com/influxdata/platform/task/backend/executor"
	"go.uber.org/zap/zaptest"
)

type fakeQueryService struct {
	mu       sync.Mutex
	queries  map[string]*fakeQuery
	queryErr error
}

var _ query.AsyncQueryService = (*fakeQueryService)(nil)

func newFakeQueryService() *fakeQueryService {
	return &fakeQueryService{queries: make(map[string]*fakeQuery)}
}

func (s *fakeQueryService) Query(ctx context.Context, orgID platform.ID, query *query.Spec) (query.Query, error) {
	panic("not implemented")
}

func (s *fakeQueryService) QueryWithCompile(ctx context.Context, orgID platform.ID, q string) (query.Query, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.queryErr != nil {
		err := s.queryErr
		s.queryErr = nil
		return nil, err
	}

	fq := &fakeQuery{
		wait:  make(chan struct{}),
		ready: make(chan map[string]query.Result),
	}
	s.queries[q] = fq
	go fq.run()

	return fq, nil
}

// SucceedQuery allows the running query matching the given script to return on its Ready channel.
func (s *fakeQueryService) SucceedQuery(script string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Unblock the query.
	close(s.queries[script].wait)
	delete(s.queries, script)
}

// FailQuery closes the running query's Ready channel and sets its error to the given value.
func (s *fakeQueryService) FailQuery(script string, forced error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Unblock the query.
	s.queries[script].forcedError = forced
	close(s.queries[script].wait)
	delete(s.queries, script)
}

// FailNextQuery causes the next call to QueryWithCompile to return the given error.
func (s *fakeQueryService) FailNextQuery(forced error) {
	s.queryErr = forced
}

// WaitForQueryLive ensures that the query has made it into the service.
// This is particularly useful for the synchronous executor,
// because the execution starts on a separate goroutine.
func (s *fakeQueryService) WaitForQueryLive(t *testing.T, script string) {
	const attempts = 10
	for i := 0; i < attempts; i++ {
		if i != 0 {
			time.Sleep(5 * time.Millisecond)
		}

		s.mu.Lock()
		_, ok := s.queries[script]
		s.mu.Unlock()
		if ok {
			return
		}
	}

	t.Fatalf("Did not see live query %q in time", script)
}

type fakeQuery struct {
	ready       chan map[string]query.Result
	wait        chan struct{} // Blocks Ready from returning.
	forcedError error         // Value to return from Err() method.
}

var _ query.Query = (*fakeQuery)(nil)

func (q *fakeQuery) Spec() *query.Spec                     { return nil }
func (q *fakeQuery) Done()                                 {}
func (q *fakeQuery) Cancel()                               {}
func (q *fakeQuery) Err() error                            { return q.forcedError }
func (q *fakeQuery) Statistics() query.Statistics          { return query.Statistics{} }
func (q *fakeQuery) Ready() <-chan map[string]query.Result { return q.ready }

// run is intended to be run on its own goroutine.
// It blocks until q.wait is closed, then sends a fake result on the q.ready channel.
func (q *fakeQuery) run() {
	// Wait for call to set query success/fail.
	<-q.wait

	if q.forcedError == nil {
		res := newFakeResult()
		q.ready <- map[string]query.Result{
			res.Name(): res,
		}
	} else {
		close(q.ready)
	}
}

// fakeResult is a dumb implementation of query.Result that always returns the same values.
type fakeResult struct {
	name  string
	table query.Table
}

var _ query.Result = (*fakeResult)(nil)

func newFakeResult() *fakeResult {
	meta := []query.ColMeta{{Label: "x", Type: query.TInt}}
	vals := []values.Value{values.NewIntValue(int64(1))}
	gk := execute.NewGroupKey(meta, vals)
	a := &execute.Allocator{Limit: math.MaxInt64}
	b := execute.NewColListTableBuilder(gk, a)
	i := b.AddCol(meta[0])
	b.AppendInt(i, int64(1))
	t, err := b.Table()
	if err != nil {
		panic(err)
	}
	return &fakeResult{name: "res", table: t}
}

func (r *fakeResult) Name() string                { return r.name }
func (r *fakeResult) Tables() query.TableIterator { return tables{r.table} }

// tables makes a TableIterator out of a slice of Tables.
type tables []query.Table

var _ query.TableIterator = tables(nil)

func (ts tables) Do(f func(query.Table) error) error {
	for _, t := range ts {
		if err := f(t); err != nil {
			return err
		}
	}
	return nil
}

type system struct {
	name string
	svc  *fakeQueryService
	st   backend.Store
	ex   backend.Executor
}

type createSysFn func(t *testing.T) *system

func createAsyncSystem(t *testing.T) *system {
	svc := newFakeQueryService()
	st := backend.NewInMemStore()
	return &system{
		name: "AsyncExecutor",
		svc:  svc,
		st:   st,
		ex:   executor.NewAsyncQueryServiceExecutor(zaptest.NewLogger(t), svc, st),
	}
}

func createSyncSystem(t *testing.T) *system {
	svc := newFakeQueryService()
	st := backend.NewInMemStore()
	return &system{
		name: "SynchronousExecutor",
		svc:  svc,
		st:   st,
		ex: executor.NewQueryServiceExecutor(
			zaptest.NewLogger(t),
			query.QueryServiceBridge{
				AsyncQueryService: svc,
			},
			st,
		),
	}
}

func TestExecutor(t *testing.T) {
	for _, fn := range []createSysFn{createAsyncSystem, createSyncSystem} {
		testExecutorQuerySuccess(t, fn)
		testExecutorQueryFailure(t, fn)
		testExecutorPromiseCancel(t, fn)
		testExecutorServiceError(t, fn)
	}
}

const testScript = `option task = {
			name: "foo",
			every: 1m,
		}
		from(bucket: "one") |> toHTTP(url: "http://example.com")`

func testExecutorQuerySuccess(t *testing.T, fn createSysFn) {
	sys := fn(t)
	t.Run(sys.name+"/QuerySuccess", func(t *testing.T) {
		tid, err := sys.st.CreateTask(context.Background(), platform.ID("org"), platform.ID("user"), testScript)
		if err != nil {
			t.Fatal(err)
		}
		qr := backend.QueuedRun{TaskID: tid, RunID: platform.ID{1}, Now: 123}
		rp, err := sys.ex.Execute(context.Background(), qr)
		if err != nil {
			t.Fatal(err)
		}

		if got := rp.Run(); !reflect.DeepEqual(qr, got) {
			t.Fatalf("unexpected queued run returned: %#v", got)
		}

		doneWaiting := make(chan struct{})
		go func() {
			_, _ = rp.Wait()
			close(doneWaiting)
		}()

		select {
		case <-doneWaiting:
			t.Fatal("Wait returned before query was unblocked")
		case <-time.After(10 * time.Millisecond):
			// Okay.
		}

		sys.svc.WaitForQueryLive(t, testScript)
		sys.svc.SucceedQuery(testScript)
		res, err := rp.Wait()
		if err != nil {
			t.Fatal(err)
		}
		if got := res.Err(); got != nil {
			t.Fatal(got)
		}

		res2, err := rp.Wait()
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(res, res2) {
			t.Fatalf("second call to wait returned a different result: %#v", res2)
		}
	})
}

func testExecutorQueryFailure(t *testing.T, fn createSysFn) {
	sys := fn(t)
	t.Run(sys.name+"/QueryFail", func(t *testing.T) {
		tid, err := sys.st.CreateTask(context.Background(), platform.ID("org"), platform.ID("user"), testScript)
		if err != nil {
			t.Fatal(err)
		}
		qr := backend.QueuedRun{TaskID: tid, RunID: platform.ID{1}, Now: 123}
		rp, err := sys.ex.Execute(context.Background(), qr)
		if err != nil {
			t.Fatal(err)
		}

		expErr := errors.New("forced error")
		sys.svc.WaitForQueryLive(t, testScript)
		sys.svc.FailQuery(testScript, expErr)
		res, err := rp.Wait()
		if err != nil {
			t.Fatal(err)
		}
		if got := res.Err(); got != expErr {
			t.Fatalf("expected error %v; got %v", expErr, got)
		}
	})
}

func testExecutorPromiseCancel(t *testing.T, fn createSysFn) {
	sys := fn(t)
	t.Run(sys.name+"/PromiseCancel", func(t *testing.T) {
		tid, err := sys.st.CreateTask(context.Background(), platform.ID("org"), platform.ID("user"), testScript)
		if err != nil {
			t.Fatal(err)
		}
		qr := backend.QueuedRun{TaskID: tid, RunID: platform.ID{1}, Now: 123}
		rp, err := sys.ex.Execute(context.Background(), qr)
		if err != nil {
			t.Fatal(err)
		}

		rp.Cancel()

		res, err := rp.Wait()
		if err != backend.ErrRunCanceled {
			t.Fatalf("expected ErrRunCanceled, got %v", err)
		}
		if res != nil {
			t.Fatalf("expected nil result after cancel, got %#v", res)
		}
	})
}

func testExecutorServiceError(t *testing.T, fn createSysFn) {
	sys := fn(t)
	t.Run(sys.name+"/ServiceError", func(t *testing.T) {
		tid, err := sys.st.CreateTask(context.Background(), platform.ID("org"), platform.ID("user"), testScript)
		if err != nil {
			t.Fatal(err)
		}
		qr := backend.QueuedRun{TaskID: tid, RunID: platform.ID{1}, Now: 123}

		var forced = errors.New("forced")
		sys.svc.FailNextQuery(forced)
		rp, err := sys.ex.Execute(context.Background(), qr)
		if err == forced {
			// Expected err matches.
			if rp != nil {
				t.Fatalf("expected nil run promise when execution fails, got %v", rp)
			}
		} else if err == nil {
			// Synchronous query service always returns <rp>, nil.
			// Make sure rp.Wait returns the correct error.
			res, err := rp.Wait()
			if err != forced {
				t.Fatalf("expected forced error, got %v", err)
			}
			if res != nil {
				t.Fatalf("expected nil result on query service error, got %v", res)
			}
		} else {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
