package functions

import (
	"fmt"
	"math"
	"sort"
	"sync"

	"github.com/influxdata/platform/query"
	"github.com/influxdata/platform/query/execute"
	"github.com/influxdata/platform/query/interpreter"
	"github.com/influxdata/platform/query/plan"
	"github.com/influxdata/platform/query/semantic"
	"github.com/influxdata/platform/query/values"
	"github.com/pkg/errors"
)

const CollateKind = "collate"

type CollateOpSpec struct {
	onething int
}

var collateSignature = semantic.FunctionSignature{
	Params: map[string]semantic.Type{
		"tables": semantic.Object,
		"on":     semantic.NewArrayType(semantic.String),
		"method": semantic.String,
	},
	ReturnType:   query.TableObjectType,
	PipeArgument: "tables",
}

func init() {
	query.RegisterFunction(CollateKind, createCollateOpSpec, collateSignature)
	query.RegisterOpSpec(CollateKind, newCollateOp)
	//TODO(nathanielc): Allow for other types of join implementations
	plan.RegisterProcedureSpec(MergeJoinKind, newCollateProcedure, JoinKind)
	execute.RegisterTransformation(MergeJoinKind, createCollateTransformation)
}

func createCollateOpSpec(args query.Arguments, a *query.Administration) (query.OperationSpec, error) {
	spec := &CollateOpSpec{}

	return spec, nil
}

func newCollateOp() query.OperationSpec {
	return new(CollateOpSpec)
}

func (s *CollateOpSpec) Kind() query.OperationKind {
	return CollateKind
}

type CollateProcedureSpec struct {
}

func newCollateProcedure(qs query.OperationSpec, pa plan.Administration) (plan.ProcedureSpec, error) {
	spec, ok := qs.(*CollateOpSpec)
	if !ok {
		return nil, fmt.Errorf("invalid spec type %T", qs)
	}

	p := &CollateProcedureSpec{}

	return p, nil
}

func (s *CollateProcedureSpec) Kind() plan.ProcedureKind {
	return MergeJoinKind
}
func (s *CollateProcedureSpec) Copy() plan.ProcedureSpec {
	ns := new(CollateProcedureSpec)

	return ns
}

// TODO: do I need this?
//func (s *CollateProcedureSpec) ParentChanged(old, new plan.ProcedureID) {
//
//}

func createCollateTransformation(id execute.DatasetID, mode execute.AccumulationMode, spec plan.ProcedureSpec, a execute.Administration) (execute.Transformation, execute.Dataset, error) {
	s, ok := spec.(*CollateProcedureSpec)
	if !ok {
		return nil, nil, fmt.Errorf("invalid spec type %T", spec)
	}

	cache := execute.NewTableBuilderCache(a.Allocator())
	d := execute.NewDataset(id, mode, cache)
	t := NewCollateTransformation(d, cache, s)
	return t, d, nil
}

type collateTransformation struct {
	d     execute.Dataset
	cache execute.TableBuilderCache
}

func NewCollateTransformation(d execute.Dataset, cache execute.TableBuilderCache, spec *CollateProcedureSpec) *collateTransformation {
	t := &collateTransformation{
		d:     d,
		cache: cache,
	}

	return t
}

func (t *collateTransformation) RetractTable(id execute.DatasetID, key query.GroupKey) error {
	panic("not implemented")
}

// Process adds a table from an incoming stream to the Join operation's data cache
func (t *collateTransformation) Process(id execute.DatasetID, tbl query.Table) error {

}

func (t *collateTransformation) UpdateWatermark(id execute.DatasetID, mark execute.Time) error {
	return t.d.UpdateWatermark(mark)
}

func (t *collateTransformation) UpdateProcessingTime(id execute.DatasetID, pt execute.Time) error {
	return t.d.UpdateProcessingTime(pt)
}

func (t *collateTransformation) Finish(id execute.DatasetID, err error) {
	t.d.Finish(err)
}
