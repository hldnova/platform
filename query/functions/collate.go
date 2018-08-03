package functions

import (
	"fmt"
	"strings"

	"github.com/influxdata/platform/query"
	"github.com/influxdata/platform/query/execute"
	"github.com/influxdata/platform/query/interpreter"
	"github.com/influxdata/platform/query/plan"
	"github.com/influxdata/platform/query/semantic"
	"github.com/influxdata/platform/query/values"
	"github.com/pkg/errors"
)

const PivotKind = "pivot"

type PivotOpSpec struct {
	RowKey   []string `json:"row_key"`
	ColKey   []string `json:"col_key"`
	ValueCol string   `json:"value_col"`
}

var pivotSignature = query.DefaultFunctionSignature()

func init() {
	pivotSignature.Params["rowKey"] = semantic.Array
	pivotSignature.Params["colKey"] = semantic.Array
	pivotSignature.Params["ValueCol"] = semantic.String

	query.RegisterFunction(PivotKind, createPivotOpSpec, pivotSignature)
	query.RegisterOpSpec(PivotKind, newPivotOp)

	plan.RegisterProcedureSpec(PivotKind, newPivotProcedure, PivotKind)
	execute.RegisterTransformation(PivotKind, createPivotTransformation)
}

func createPivotOpSpec(args query.Arguments, a *query.Administration) (query.OperationSpec, error) {
	spec := &PivotOpSpec{}

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

func newPivotOp() query.OperationSpec {
	return new(PivotOpSpec)
}

func (s *PivotOpSpec) Kind() query.OperationKind {
	return PivotKind
}

type PivotProcedureSpec struct {
	RowKey   []string
	ColKey   []string
	ValueCol string
}

func newPivotProcedure(qs query.OperationSpec, pa plan.Administration) (plan.ProcedureSpec, error) {
	spec, ok := qs.(*PivotOpSpec)
	if !ok {
		return nil, fmt.Errorf("invalid spec type %T", qs)
	}

	p := &PivotProcedureSpec{
		RowKey:   spec.RowKey,
		ColKey:   spec.ColKey,
		ValueCol: spec.ValueCol,
	}

	return p, nil
}

func (s *PivotProcedureSpec) Kind() plan.ProcedureKind {
	return PivotKind
}
func (s *PivotProcedureSpec) Copy() plan.ProcedureSpec {
	ns := new(PivotProcedureSpec)
	ns.RowKey = make([]string, len(s.RowKey))
	copy(ns.RowKey, s.RowKey)
	ns.ColKey = make([]string, len(s.ColKey))
	copy(ns.ColKey, s.ColKey)
	ns.ValueCol = s.ValueCol
	return ns
}

func createPivotTransformation(id execute.DatasetID, mode execute.AccumulationMode, spec plan.ProcedureSpec, a execute.Administration) (execute.Transformation, execute.Dataset, error) {
	s, ok := spec.(*PivotProcedureSpec)
	if !ok {
		return nil, nil, fmt.Errorf("invalid spec type %T", spec)
	}

	cache := execute.NewTableBuilderCache(a.Allocator())
	d := execute.NewDataset(id, mode, cache)
	t := NewPivotTransformation(d, cache, s)
	return t, d, nil
}

type pivotTransformation struct {
	d     execute.Dataset
	cache execute.TableBuilderCache
	spec  PivotProcedureSpec
}

func NewPivotTransformation(d execute.Dataset, cache execute.TableBuilderCache, spec *PivotProcedureSpec) *pivotTransformation {
	t := &pivotTransformation{
		d:     d,
		cache: cache,
		spec:  *spec,
	}
	return t
}

func (t *pivotTransformation) RetractTable(id execute.DatasetID, key query.GroupKey) error {
	return t.d.RetractTable(key)
}

func (t *pivotTransformation) Process(id execute.DatasetID, tbl query.Table) error {

	colsToRemove := make(map[string]bool)
	colsToRemove[t.spec.ValueCol] = true
	for _, v := range t.spec.ColKey {
		colsToRemove[v] = false
	}

	cols := make([]query.ColMeta, 0, len(tbl.Cols()))
	keyCols := make([]query.ColMeta, 0, len(tbl.Key().Cols()))
	keyValues := make([]values.Value, 0, len(tbl.Key().Cols()))
	newIDX := 0
	colMap := make([]int, 0, len(tbl.Cols()))

	for colIDX, v := range tbl.Cols() {
		if _, ok := colsToRemove[v.Label]; !ok {
			cols = append(cols, tbl.Cols()[colIDX])
			colMap[colIDX] = newIDX
			newIDX++
			if tbl.Key().HasCol(v.Label) {
				keyCols = append(keyCols, tbl.Cols()[colIDX])
				keyValues = append(keyValues, tbl.Key().Value(colIDX))
			}
		} else {
			colsToRemove[v.Label] = true
		}
	}

	for k, v := range colsToRemove {
		if !v {
			return fmt.Errorf("specified column does not exist in table: %v", k)
		}
	}

	for _, v := range t.spec.RowKey {
		if !execute.HasCol(v, tbl.Cols()) {
			return fmt.Errorf("specified column does not exist in table: %v", v)
		}
	}

	newKey := execute.NewGroupKey(keyCols, keyValues)
	builder, created := t.cache.TableBuilder(newKey)
	if !created {
		return fmt.Errorf("pivot found duplicate table with key %v", tbl.Key())
	}

	for _, c := range cols {
		builder.AddCol(c)
	}

	// at this point, we know: the group key, the existing columns we'll keep. We have a colMap
	// so we can quickly copy over values that aren't part of the pivot.
	// we know the pivot column

	keyColPrefix := strings.Join(t.spec.ColKey, "_")

	tbl.Do(func(cr query.ColReader) error {
		cr.Strings()
		return nil
	})

	execute.AppendKeyValues(newKey, builder)
	builder.AppendBools()
	return nil
}

func (t *pivotTransformation) UpdateWatermark(id execute.DatasetID, mark execute.Time) error {
	return t.d.UpdateWatermark(mark)
}

func (t *pivotTransformation) UpdateProcessingTime(id execute.DatasetID, pt execute.Time) error {
	return t.d.UpdateProcessingTime(pt)
}

func (t *pivotTransformation) Finish(id execute.DatasetID, err error) {

	t.d.Finish(err)
}
