package spectests

import (
	"time"

	"github.com/influxdata/influxql"
	"github.com/influxdata/platform/query"
	"github.com/influxdata/platform/query/ast"
	"github.com/influxdata/platform/query/execute"
	"github.com/influxdata/platform/query/functions"
	"github.com/influxdata/platform/query/semantic"
)

func init() {
	RegisterFixture(
		NewFixture(
			`SELECT mean(value), max(value) FROM db0..cpu`,
			&query.Spec{
				Operations: []*query.Operation{
					{
						ID: "from0",
						Spec: &functions.FromOpSpec{
							BucketID: bucketID,
						},
					},
					{
						ID: "range0",
						Spec: &functions.RangeOpSpec{
							Start:    query.Time{Absolute: time.Unix(0, influxql.MinTime)},
							Stop:     query.Time{Absolute: time.Unix(0, influxql.MaxTime)},
							TimeCol:  execute.DefaultTimeColLabel,
							StartCol: execute.DefaultStartColLabel,
							StopCol:  execute.DefaultStopColLabel,
						},
					},
					{
						ID: "filter0",
						Spec: &functions.FilterOpSpec{
							Fn: &semantic.FunctionExpression{
								Params: []*semantic.FunctionParam{
									{Key: &semantic.Identifier{Name: "r"}},
								},
								Body: &semantic.LogicalExpression{
									Operator: ast.AndOperator,
									Left: &semantic.BinaryExpression{
										Operator: ast.EqualOperator,
										Left: &semantic.MemberExpression{
											Object: &semantic.IdentifierExpression{
												Name: "r",
											},
											Property: "_measurement",
										},
										Right: &semantic.StringLiteral{
											Value: "cpu",
										},
									},
									Right: &semantic.BinaryExpression{
										Operator: ast.EqualOperator,
										Left: &semantic.MemberExpression{
											Object: &semantic.IdentifierExpression{
												Name: "r",
											},
											Property: "_field",
										},
										Right: &semantic.StringLiteral{
											Value: "value",
										},
									},
								},
							},
						},
					},
					{
						ID: "group0",
						Spec: &functions.GroupOpSpec{
							By: []string{"_measurement", "_start"},
						},
					},
					{
						ID: "mean0",
						Spec: &functions.MeanOpSpec{
							AggregateConfig: execute.AggregateConfig{
								TimeSrc: execute.DefaultStartColLabel,
								TimeDst: execute.DefaultTimeColLabel,
								Columns: []string{execute.DefaultValueColLabel},
							},
						},
					},
					{
						ID: "from1",
						Spec: &functions.FromOpSpec{
							BucketID: bucketID,
						},
					},
					{
						ID: "range1",
						Spec: &functions.RangeOpSpec{
							Start:    query.Time{Absolute: time.Unix(0, influxql.MinTime)},
							Stop:     query.Time{Absolute: time.Unix(0, influxql.MaxTime)},
							TimeCol:  execute.DefaultTimeColLabel,
							StartCol: execute.DefaultStartColLabel,
							StopCol:  execute.DefaultStopColLabel,
						},
					},
					{
						ID: "filter1",
						Spec: &functions.FilterOpSpec{
							Fn: &semantic.FunctionExpression{
								Params: []*semantic.FunctionParam{
									{Key: &semantic.Identifier{Name: "r"}},
								},
								Body: &semantic.LogicalExpression{
									Operator: ast.AndOperator,
									Left: &semantic.BinaryExpression{
										Operator: ast.EqualOperator,
										Left: &semantic.MemberExpression{
											Object: &semantic.IdentifierExpression{
												Name: "r",
											},
											Property: "_measurement",
										},
										Right: &semantic.StringLiteral{
											Value: "cpu",
										},
									},
									Right: &semantic.BinaryExpression{
										Operator: ast.EqualOperator,
										Left: &semantic.MemberExpression{
											Object: &semantic.IdentifierExpression{
												Name: "r",
											},
											Property: "_field",
										},
										Right: &semantic.StringLiteral{
											Value: "value",
										},
									},
								},
							},
						},
					},
					{
						ID: "group1",
						Spec: &functions.GroupOpSpec{
							By: []string{"_measurement", "_start"},
						},
					},
					{
						ID: "max0",
						Spec: &functions.MaxOpSpec{
							SelectorConfig: execute.SelectorConfig{
								Column: execute.DefaultValueColLabel,
							},
						},
					},
					{
						ID: "join0",
						Spec: &functions.JoinOpSpec{
							On: []string{"_measurement"},
							Fn: &semantic.FunctionExpression{
								Params: []*semantic.FunctionParam{{
									Key: &semantic.Identifier{Name: "tables"},
								}},
								Body: &semantic.ObjectExpression{
									Properties: []*semantic.Property{
										{
											Key: &semantic.Identifier{Name: "val0"},
											Value: &semantic.MemberExpression{
												Object: &semantic.IdentifierExpression{
													Name: "tables",
												},
												Property: "t0",
											},
										},
										{
											Key: &semantic.Identifier{Name: "val1"},
											Value: &semantic.MemberExpression{
												Object: &semantic.IdentifierExpression{
													Name: "tables",
												},
												Property: "t1",
											},
										},
									},
								},
							},
							TableNames: map[query.OperationID]string{
								"mean0": "t0",
								"max0":  "t1",
							},
						},
					},
					{
						ID: "map0",
						Spec: &functions.MapOpSpec{
							Fn: &semantic.FunctionExpression{
								Params: []*semantic.FunctionParam{{
									Key: &semantic.Identifier{Name: "r"},
								}},
								Body: &semantic.ObjectExpression{
									Properties: []*semantic.Property{
										{
											Key: &semantic.Identifier{Name: "_time"},
											Value: &semantic.MemberExpression{
												Object: &semantic.IdentifierExpression{
													Name: "r",
												},
												Property: "_time",
											},
										},
										{
											Key: &semantic.Identifier{Name: "mean"},
											Value: &semantic.MemberExpression{
												Object: &semantic.IdentifierExpression{
													Name: "r",
												},
												Property: "val0",
											},
										},
										{
											Key: &semantic.Identifier{Name: "max"},
											Value: &semantic.MemberExpression{
												Object: &semantic.IdentifierExpression{
													Name: "r",
												},
												Property: "val1",
											},
										},
									},
								},
							},
							MergeKey: true,
						},
					},
					{
						ID: "yield0",
						Spec: &functions.YieldOpSpec{
							Name: "0",
						},
					},
				},
				Edges: []query.Edge{
					{Parent: "from0", Child: "range0"},
					{Parent: "range0", Child: "filter0"},
					{Parent: "filter0", Child: "group0"},
					{Parent: "group0", Child: "mean0"},
					{Parent: "from1", Child: "range1"},
					{Parent: "range1", Child: "filter1"},
					{Parent: "filter1", Child: "group1"},
					{Parent: "group1", Child: "max0"},
					{Parent: "mean0", Child: "join0"},
					{Parent: "max0", Child: "join0"},
					{Parent: "join0", Child: "map0"},
					{Parent: "map0", Child: "yield0"},
				},
				Now: Now(),
			},
		),
	)
}
