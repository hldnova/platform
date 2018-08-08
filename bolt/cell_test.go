package bolt_test

import (
	"context"
	"testing"

	"github.com/influxdata/platform"
	platformtesting "github.com/influxdata/platform/testing"
)

func initCellService(f platformtesting.CellFields, t *testing.T) (platform.CellService, func()) {
	c, closeFn, err := NewTestClient()
	if err != nil {
		t.Fatalf("failed to create new bolt client: %v", err)
	}
	c.IDGenerator = f.IDGenerator
	ctx := context.TODO()
	for _, b := range f.Cells {
		if err := c.PutCell(ctx, b); err != nil {
			t.Fatalf("failed to populate cells")
		}
	}
	return c, func() {
		defer closeFn()
		for _, b := range f.Cells {
			if err := c.DeleteCell(ctx, b.ID); err != nil {
				t.Logf("failed to remove cell: %v", err)
			}
		}
	}
}

func TestCellService_CreateCell(t *testing.T) {
	platformtesting.CreateCell(initCellService, t)
}

func TestCellService_FindCellByID(t *testing.T) {
	platformtesting.FindCellByID(initCellService, t)
}

func TestCellService_FindCells(t *testing.T) {
	platformtesting.FindCells(initCellService, t)
}

func TestCellService_DeleteCell(t *testing.T) {
	platformtesting.DeleteCell(initCellService, t)
}

func TestCellService_UpdateCell(t *testing.T) {
	platformtesting.UpdateCell(initCellService, t)
}
