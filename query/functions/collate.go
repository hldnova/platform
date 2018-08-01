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
	RowKey   []string `json:"row_key"`
	ColKey   []string `json:"col_key"`
	ValueCol string   `json:"value_col"`
}

var collateSignature = query.DefaultFunctionSignature()

func init() {
	collateSignature.Params["rowKey"] = semantic.Array
	collateSignature.Params["colKey"] = semantic.Array
	collateSignature.Params["ValueCol"] = semantic.String

	query.RegisterFunction(CollateKind, createCollateOpSpec, collateSignature)
	query.RegisterOpSpec(CollateKind, newCollateOp)

	plan.RegisterProcedureSpec(CollateKind, newCollateProcedure, CollateKind)
	execute.RegisterTransformation(CollateKind, createCollateTransformation)
}

func createCollateOpSpec(args query.Arguments, a *query.Administration) (query.OperationSpec, error) {
	spec := &CollateOpSpec{}

	array, err := args.GetRequiredArray("rowKey", semantic.String)
	if err != nil {
		return nil, err
	}

	spec.RowKey, err = interpreter.ToStringArray(array)
	if err != nil {
		return nil, err
	}

	array, err = args.GetRequiredArray("colKey", semantic.String)
	if err != nil {
		return nil, err
	}

	spec.ColKey, err = interpreter.ToStringArray(array)
	if err != nil {
		return nil, err
	}

	valueCol, err := args.GetRequiredString("ValueCol")
	if err != nil {
		return nil, err
	}
	spec.ValueCol = valueCol

	return spec, nil
}

func newCollateOp() query.OperationSpec {
	return new(CollateOpSpec)
}

func (s *CollateOpSpec) Kind() query.OperationKind {
	return CollateKind
}

type CollateProcedureSpec struct {
	RowKey   []string
	ColKey   []string
	ValueCol string
}

func newCollateProcedure(qs query.OperationSpec, pa plan.Administration) (plan.ProcedureSpec, error) {
	spec, ok := qs.(*CollateOpSpec)
	if !ok {
		return nil, fmt.Errorf("invalid spec type %T", qs)
	}

	p := &CollateProcedureSpec{
		RowKey:   spec.RowKey,
		ColKey:   spec.ColKey,
		ValueCol: spec.ValueCol,
	}

	return p, nil
}

func (s *CollateProcedureSpec) Kind() plan.ProcedureKind {
	return CollateKind
}
func (s *CollateProcedureSpec) Copy() plan.ProcedureSpec {
	ns := new(CollateProcedureSpec)
	ns.RowKey = make([]string, len(s.RowKey))
	copy(ns.RowKey, s.RowKey)
	ns.ColKey = make([]string, len(s.ColKey))
	copy(ns.ColKey, s.ColKey)
	ns.ValueCol = s.ValueCol
	return ns
}

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
	spec  CollateProcedureSpec
}

func NewCollateTransformation(d execute.Dataset, cache execute.TableBuilderCache, spec *CollateProcedureSpec) *collateTransformation {
	t := &collateTransformation{
		d:     d,
		cache: cache,
		spec:  *spec,
	}
	return t
}

func (t *collateTransformation) RetractTable(id execute.DatasetID, key query.GroupKey) error {
	return t.d.RetractTable(key)
}

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
