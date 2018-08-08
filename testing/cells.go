package testing

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/influxdata/platform"
	"github.com/influxdata/platform/mock"
)

const (
	cellOneID   = "020f755c3c082000"
	cellTwoID   = "020f755c3c082001"
	cellThreeID = "020f755c3c082002"
)

var cellCmpOptions = cmp.Options{
	cmp.Comparer(func(x, y []byte) bool {
		return bytes.Equal(x, y)
	}),
	cmp.Transformer("Sort", func(in []*platform.Cell) []*platform.Cell {
		out := append([]*platform.Cell(nil), in...) // Copy input to avoid mutating it
		sort.Slice(out, func(i, j int) bool {
			return out[i].ID.String() > out[j].ID.String()
		})
		return out
	}),
}

// CellFields will include the IDGenerator, and cells
type CellFields struct {
	IDGenerator platform.IDGenerator
	Cells       []*platform.Cell
}

// CreateCell testing
func CreateCell(
	init func(CellFields, *testing.T) (platform.CellService, func()),
	t *testing.T,
) {
	type args struct {
		cell *platform.Cell
	}
	type wants struct {
		err   error
		cells []*platform.Cell
	}

	tests := []struct {
		name   string
		fields CellFields
		args   args
		wants  wants
	}{
		{
			name: "basic create cell",
			fields: CellFields{
				IDGenerator: &mock.IDGenerator{
					IDFn: func() platform.ID {
						return idFromString(t, cellTwoID)
					},
				},
				Cells: []*platform.Cell{
					{
						CellContents: platform.CellContents{
							ID:   idFromString(t, cellOneID),
							Name: "cell1",
						},
					},
				},
			},
			args: args{
				cell: &platform.Cell{
					CellContents: platform.CellContents{
						Name: "cell2",
					},
					Visualization: platform.V1Visualization{
						TimeFormat: "rfc3339",
					},
				},
			},
			wants: wants{
				cells: []*platform.Cell{
					{
						CellContents: platform.CellContents{
							ID:   idFromString(t, cellOneID),
							Name: "cell1",
						},
						Visualization: platform.EmptyVisualization{},
					},
					{
						CellContents: platform.CellContents{
							ID:   idFromString(t, cellTwoID),
							Name: "cell2",
						},
						Visualization: platform.V1Visualization{
							TimeFormat: "rfc3339",
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, done := init(tt.fields, t)
			defer done()
			ctx := context.TODO()
			err := s.CreateCell(ctx, tt.args.cell)
			if (err != nil) != (tt.wants.err != nil) {
				t.Fatalf("expected error '%v' got '%v'", tt.wants.err, err)
			}

			if err != nil && tt.wants.err != nil {
				if err.Error() != tt.wants.err.Error() {
					t.Fatalf("expected error messages to match '%v' got '%v'", tt.wants.err, err.Error())
				}
			}
			defer s.DeleteCell(ctx, tt.args.cell.ID)

			cells, _, err := s.FindCells(ctx, platform.CellFilter{})
			if err != nil {
				t.Fatalf("failed to retrieve cells: %v", err)
			}
			if diff := cmp.Diff(cells, tt.wants.cells, cellCmpOptions...); diff != "" {
				t.Errorf("cells are different -got/+want\ndiff %s", diff)
			}
		})
	}
}

// FindCellByID testing
func FindCellByID(
	init func(CellFields, *testing.T) (platform.CellService, func()),
	t *testing.T,
) {
	type args struct {
		id platform.ID
	}
	type wants struct {
		err  error
		cell *platform.Cell
	}

	tests := []struct {
		name   string
		fields CellFields
		args   args
		wants  wants
	}{
		{
			name: "basic find cell by id",
			fields: CellFields{
				Cells: []*platform.Cell{
					{
						CellContents: platform.CellContents{
							ID:   idFromString(t, cellOneID),
							Name: "cell1",
						},
						Visualization: platform.EmptyVisualization{},
					},
					{
						CellContents: platform.CellContents{
							ID:   idFromString(t, cellTwoID),
							Name: "cell2",
						},
						Visualization: platform.V1Visualization{
							TimeFormat: "rfc3339",
						},
					},
				},
			},
			args: args{
				id: idFromString(t, cellTwoID),
			},
			wants: wants{
				cell: &platform.Cell{
					CellContents: platform.CellContents{
						ID:   idFromString(t, cellTwoID),
						Name: "cell2",
					},
					Visualization: platform.V1Visualization{
						TimeFormat: "rfc3339",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, done := init(tt.fields, t)
			defer done()
			ctx := context.TODO()

			cell, err := s.FindCellByID(ctx, tt.args.id)
			if (err != nil) != (tt.wants.err != nil) {
				t.Fatalf("expected errors to be equal '%v' got '%v'", tt.wants.err, err)
			}

			if err != nil && tt.wants.err != nil {
				if err.Error() != tt.wants.err.Error() {
					t.Fatalf("expected error '%v' got '%v'", tt.wants.err, err)
				}
			}

			if diff := cmp.Diff(cell, tt.wants.cell, cellCmpOptions...); diff != "" {
				t.Errorf("cell is different -got/+want\ndiff %s", diff)
			}
		})
	}
}

// FindCells testing
func FindCells(
	init func(CellFields, *testing.T) (platform.CellService, func()),
	t *testing.T,
) {
	type args struct {
		ID   platform.ID
		name string
	}

	type wants struct {
		cells []*platform.Cell
		err   error
	}
	tests := []struct {
		name   string
		fields CellFields
		args   args
		wants  wants
	}{
		{
			name: "find all cells",
			fields: CellFields{
				Cells: []*platform.Cell{
					{
						CellContents: platform.CellContents{
							ID:   idFromString(t, cellOneID),
							Name: "cell1",
						},
						Visualization: platform.EmptyVisualization{},
					},
					{
						CellContents: platform.CellContents{
							ID:   idFromString(t, cellTwoID),
							Name: "cell2",
						},
						Visualization: platform.V1Visualization{
							TimeFormat: "rfc3339",
						},
					},
				},
			},
			args: args{},
			wants: wants{
				cells: []*platform.Cell{
					{
						CellContents: platform.CellContents{
							ID:   idFromString(t, cellOneID),
							Name: "cell1",
						},
						Visualization: platform.EmptyVisualization{},
					},
					{
						CellContents: platform.CellContents{
							ID:   idFromString(t, cellTwoID),
							Name: "cell2",
						},
						Visualization: platform.V1Visualization{
							TimeFormat: "rfc3339",
						},
					},
				},
			},
		},
		{
			name: "find cell by id",
			fields: CellFields{
				Cells: []*platform.Cell{
					{
						CellContents: platform.CellContents{
							ID:   idFromString(t, cellOneID),
							Name: "cell1",
						},
						Visualization: platform.EmptyVisualization{},
					},
					{
						CellContents: platform.CellContents{
							ID:   idFromString(t, cellTwoID),
							Name: "cell2",
						},
						Visualization: platform.V1Visualization{
							TimeFormat: "rfc3339",
						},
					},
				},
			},
			args: args{
				ID: idFromString(t, cellTwoID),
			},
			wants: wants{
				cells: []*platform.Cell{
					{
						CellContents: platform.CellContents{
							ID:   idFromString(t, cellTwoID),
							Name: "cell2",
						},
						Visualization: platform.V1Visualization{
							TimeFormat: "rfc3339",
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, done := init(tt.fields, t)
			defer done()
			ctx := context.TODO()

			filter := platform.CellFilter{}
			if tt.args.ID != nil {
				filter.ID = &tt.args.ID
			}

			cells, _, err := s.FindCells(ctx, filter)
			if (err != nil) != (tt.wants.err != nil) {
				t.Fatalf("expected errors to be equal '%v' got '%v'", tt.wants.err, err)
			}

			if err != nil && tt.wants.err != nil {
				if err.Error() != tt.wants.err.Error() {
					t.Fatalf("expected error '%v' got '%v'", tt.wants.err, err)
				}
			}

			if diff := cmp.Diff(cells, tt.wants.cells, cellCmpOptions...); diff != "" {
				t.Errorf("cells are different -got/+want\ndiff %s", diff)
			}
		})
	}
}

// DeleteCell testing
func DeleteCell(
	init func(CellFields, *testing.T) (platform.CellService, func()),
	t *testing.T,
) {
	type args struct {
		ID platform.ID
	}
	type wants struct {
		err   error
		cells []*platform.Cell
	}

	tests := []struct {
		name   string
		fields CellFields
		args   args
		wants  wants
	}{
		{
			name: "delete cells using exist id",
			fields: CellFields{
				Cells: []*platform.Cell{
					{
						CellContents: platform.CellContents{
							ID:   idFromString(t, cellOneID),
							Name: "cell1",
						},
						Visualization: platform.EmptyVisualization{},
					},
					{
						CellContents: platform.CellContents{
							ID:   idFromString(t, cellTwoID),
							Name: "cell2",
						},
						Visualization: platform.V1Visualization{
							TimeFormat: "rfc3339",
						},
					},
				},
			},
			args: args{
				ID: idFromString(t, cellOneID),
			},
			wants: wants{
				cells: []*platform.Cell{
					{
						CellContents: platform.CellContents{
							ID:   idFromString(t, cellTwoID),
							Name: "cell2",
						},
						Visualization: platform.V1Visualization{
							TimeFormat: "rfc3339",
						},
					},
				},
			},
		},
		{
			name: "delete cells using id that does not exist",
			fields: CellFields{
				Cells: []*platform.Cell{
					{
						CellContents: platform.CellContents{
							ID:   idFromString(t, cellOneID),
							Name: "cell1",
						},
						Visualization: platform.EmptyVisualization{},
					},
					{
						CellContents: platform.CellContents{
							ID:   idFromString(t, cellTwoID),
							Name: "cell2",
						},
						Visualization: platform.V1Visualization{
							TimeFormat: "rfc3339",
						},
					},
				},
			},
			args: args{
				ID: idFromString(t, cellThreeID),
			},
			wants: wants{
				err: fmt.Errorf("cell not found"),
				cells: []*platform.Cell{
					{
						CellContents: platform.CellContents{
							ID:   idFromString(t, cellOneID),
							Name: "cell1",
						},
						Visualization: platform.EmptyVisualization{},
					},
					{
						CellContents: platform.CellContents{
							ID:   idFromString(t, cellTwoID),
							Name: "cell2",
						},
						Visualization: platform.V1Visualization{
							TimeFormat: "rfc3339",
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, done := init(tt.fields, t)
			defer done()
			ctx := context.TODO()
			err := s.DeleteCell(ctx, tt.args.ID)
			if (err != nil) != (tt.wants.err != nil) {
				t.Fatalf("expected error '%v' got '%v'", tt.wants.err, err)
			}

			if err != nil && tt.wants.err != nil {
				if err.Error() != tt.wants.err.Error() {
					t.Fatalf("expected error messages to match '%v' got '%v'", tt.wants.err, err.Error())
				}
			}

			filter := platform.CellFilter{}
			cells, _, err := s.FindCells(ctx, filter)
			if err != nil {
				t.Fatalf("failed to retrieve cells: %v", err)
			}
			if diff := cmp.Diff(cells, tt.wants.cells, cellCmpOptions...); diff != "" {
				t.Errorf("cells are different -got/+want\ndiff %s", diff)
			}
		})
	}
}

// UpdateCell testing
func UpdateCell(
	init func(CellFields, *testing.T) (platform.CellService, func()),
	t *testing.T,
) {
	type args struct {
		name          string
		visualization platform.Visualization
		id            platform.ID
	}
	type wants struct {
		err  error
		cell *platform.Cell
	}

	tests := []struct {
		name   string
		fields CellFields
		args   args
		wants  wants
	}{
		{
			name: "update name",
			fields: CellFields{
				Cells: []*platform.Cell{
					{
						CellContents: platform.CellContents{
							ID:   idFromString(t, cellOneID),
							Name: "cell1",
						},
						Visualization: platform.EmptyVisualization{},
					},
					{
						CellContents: platform.CellContents{
							ID:   idFromString(t, cellTwoID),
							Name: "cell2",
						},
						Visualization: platform.V1Visualization{
							TimeFormat: "rfc3339",
						},
					},
				},
			},
			args: args{
				id:   idFromString(t, cellOneID),
				name: "changed",
			},
			wants: wants{
				cell: &platform.Cell{
					CellContents: platform.CellContents{
						ID:   idFromString(t, cellOneID),
						Name: "changed",
					},
					Visualization: platform.EmptyVisualization{},
				},
			},
		},
		{
			name: "update visualization",
			fields: CellFields{
				Cells: []*platform.Cell{
					{
						CellContents: platform.CellContents{
							ID:   idFromString(t, cellOneID),
							Name: "cell1",
						},
						Visualization: platform.EmptyVisualization{},
					},
					{
						CellContents: platform.CellContents{
							ID:   idFromString(t, cellTwoID),
							Name: "cell2",
						},
						Visualization: platform.V1Visualization{
							TimeFormat: "rfc3339",
						},
					},
				},
			},
			args: args{
				id: idFromString(t, cellOneID),
				visualization: platform.V1Visualization{
					TimeFormat: "rfc3339",
				},
			},
			wants: wants{
				cell: &platform.Cell{
					CellContents: platform.CellContents{
						ID:   idFromString(t, cellOneID),
						Name: "cell1",
					},
					Visualization: platform.V1Visualization{
						TimeFormat: "rfc3339",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, done := init(tt.fields, t)
			defer done()
			ctx := context.TODO()

			upd := platform.CellUpdate{}
			if tt.args.name != "" {
				upd.Name = &tt.args.name
			}
			if tt.args.visualization != nil {
				upd.Visualization = tt.args.visualization
			}

			cell, err := s.UpdateCell(ctx, tt.args.id, upd)
			if (err != nil) != (tt.wants.err != nil) {
				t.Fatalf("expected error '%v' got '%v'", tt.wants.err, err)
			}

			if err != nil && tt.wants.err != nil {
				if err.Error() != tt.wants.err.Error() {
					t.Fatalf("expected error messages to match '%v' got '%v'", tt.wants.err, err.Error())
				}
			}

			if diff := cmp.Diff(cell, tt.wants.cell, cellCmpOptions...); diff != "" {
				t.Errorf("cell is different -got/+want\ndiff %s", diff)
			}
		})
	}
}
