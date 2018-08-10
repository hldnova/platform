package plan

import (
	"fmt"
	"time"

	"github.com/influxdata/platform/query"
	uuid "github.com/satori/go.uuid"
)

type ProcedureID uuid.UUID

func (id ProcedureID) String() string {
	return uuid.UUID(id).String()
}

var ZeroProcedureID ProcedureID

type Procedure struct {
	plan     *PlanSpec
	ID       ProcedureID
	Parents  []ProcedureID
	Children []ProcedureID
	Spec     ProcedureSpec
	Bounds   BoundsSpec
}

func (p *Procedure) Copy() *Procedure {
	np := new(Procedure)
	np.ID = p.ID

	np.plan = p.plan

	np.Parents = make([]ProcedureID, len(p.Parents))
	copy(np.Parents, p.Parents)

	np.Children = make([]ProcedureID, len(p.Children))
	copy(np.Children, p.Children)

	np.Spec = p.Spec.Copy()

	return np
}

func (p *Procedure) DoChildren(f func(pr *Procedure)) {
	for _, id := range p.Children {
		f(p.plan.Procedures[id])
	}
}
func (p *Procedure) DoParents(f func(pr *Procedure)) {
	for _, id := range p.Parents {
		f(p.plan.Procedures[id])
	}
}
func (p *Procedure) Child(i int) *Procedure {
	return p.plan.Procedures[p.Children[i]]
}

type Administration interface {
	ConvertID(query.OperationID) ProcedureID
}

type CreateProcedureSpec func(query.OperationSpec, Administration) (ProcedureSpec, error)

// ProcedureSpec specifies an operation as part of a query.
type ProcedureSpec interface {
	// Kind returns the kind of the procedure.
	Kind() ProcedureKind
	Copy() ProcedureSpec
}

type PushDownProcedureSpec interface {
	PushDownRules() []PushDownRule
	PushDown(root *Procedure, dup func() *Procedure)
}

type BoundedProcedureSpec interface {
	TimeBounds() BoundsSpec
}

type YieldProcedureSpec interface {
	YieldName() string
}
type AggregateProcedureSpec interface {
	// AggregateMethod specifies which aggregate method to push down to the storage layer.
	AggregateMethod() string
	// ReAggregateSpec specifies an aggregate procedure to use when aggregating the individual pushed down results.
	ReAggregateSpec() ProcedureSpec
}

type ParentAwareProcedureSpec interface {
	ParentChanged(old, new ProcedureID)
}

// TODO(nathanielc): make this more formal using commute/associative properties
type PushDownRule struct {
	Root    ProcedureKind
	Through []ProcedureKind
	Match   func(ProcedureSpec) bool
}

// ProcedureKind denotes the kind of operations.
type ProcedureKind string

type BoundsSpec struct {
	Start query.Time
	Stop  query.Time
}

// [-3, 0]
// [-2, 2]

func (b BoundsSpec) Union(o BoundsSpec, now time.Time) (u BoundsSpec) {
	u.Start = b.Start
	if u.Start.IsZero() || (!o.Start.IsZero() && o.Start.Time(now).Before(b.Start.Time(now))) {
		u.Start = o.Start
	}
	u.Stop = b.Stop
	if u.Stop.IsZero() || (!o.Start.IsZero() && o.Stop.Time(now).After(b.Stop.Time(now))) {
		u.Stop = o.Stop
	}
	return
}

// NOTE: if either b.Start or o.Start are both relative or non-zero, and b.Stop or o.Stop are non-relative zero,
// It is assumed that b.Stop and/or o.Stop are meant to be relative zero (i.e., `now`) to make the logic work for all cases.
// Ex: b = [now - 1h, Unix(0)], o = [now - 2h, Unix(0)]
// in this case, the upper bound of 0 for both b and o becomes `now`. So the bounds are actually:
// [now - 1h, now], [now - 2h, now], and we can get the expected result of [now - 1h, now]

// Intersect returns the intersection of two bounds. If there is no intersection,
// the first bounds are returned.
func (b BoundsSpec) Intersect(o BoundsSpec, now time.Time) (i BoundsSpec) {
	var bStop query.Time
	if !b.Start.IsZero() && b.Stop.IsZero() {
		bStop.IsRelative = true
	} else {
		bStop = b.Stop
	}

	var oStop query.Time
	if !o.Start.IsZero() && o.Stop.IsZero() {
		oStop.IsRelative = true
	} else {
		oStop = o.Stop
	}

	if (b.Start.IsZero() || (o.Start.Time(now).After(b.Start.Time(now)))) &&
		(o.Start.Time(now).Before(bStop.Time(now))) {
		i.Start = o.Start
	} else {
		i.Start = b.Start
	}

	if oStop.Time(now).Before(bStop.Time(now)) &&
		(o.Stop.Time(now).After(b.Start.Time(now))) {
		i.Stop = o.Stop
	} else {
		i.Stop = b.Stop
	}

	return
}

type WindowSpec struct {
	Every  query.Duration
	Period query.Duration
	Round  query.Duration
	Start  query.Time
}

var kindToProcedure = make(map[ProcedureKind]CreateProcedureSpec)
var queryOpToProcedure = make(map[query.OperationKind][]CreateProcedureSpec)

// RegisterProcedureSpec registers a new procedure with the specified kind.
// The call panics if the kind is not unique.
func RegisterProcedureSpec(k ProcedureKind, c CreateProcedureSpec, qks ...query.OperationKind) {
	if kindToProcedure[k] != nil {
		panic(fmt.Errorf("duplicate registration for procedure kind %v", k))
	}
	kindToProcedure[k] = c
	for _, qk := range qks {
		queryOpToProcedure[qk] = append(queryOpToProcedure[qk], c)
	}
}
