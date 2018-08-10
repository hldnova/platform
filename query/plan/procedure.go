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

func (b BoundsSpec) Union(o BoundsSpec, now time.Time) (u BoundsSpec) {
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

	if b.Start.IsZero() || (!o.Start.IsZero() && o.Start.Time(now).Before(b.Start.Time(now))) {
		u.Start = o.Start
	} else {
		u.Start = b.Start
	}

	if bStop.IsZero() || (oStop.Time(now).After(bStop.Time(now))) {
		u.Stop = o.Stop
	} else {
		u.Stop = b.Stop
	}

	return
}

// Intersect returns the intersection of two bounds. If there is no intersection,
// the first bounds are returned. If start is not set, the start bounds of the other provided bounds are
// automatically used. If stop is not set, it is assumed to be the current time for the purposes of
// logic.
func (b BoundsSpec) Intersect(o BoundsSpec, now time.Time) (i BoundsSpec) {
	// bStop and oStop are temporary adjusted variables to be used in this function
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

	if b.Start.IsZero() || (o.Start.Time(now).After(b.Start.Time(now))) &&
		(o.Start.Time(now).Before(bStop.Time(now))) {
		i.Start = o.Start
	} else {
		i.Start = b.Start
	}

	if bStop.IsZero() || oStop.Time(now).Before(bStop.Time(now)) &&
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
