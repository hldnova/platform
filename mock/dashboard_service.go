package mock

import (
	"context"

	"github.com/influxdata/platform"
)

var _ platform.DashboardService = &DashboardService{}

type DashboardService struct {
	CreateDashboardF   func(context.Context, *platform.Dashboard) error
	FindDashboardByIDF func(context.Context, platform.ID) (*platform.Dashboard, error)
	FindDashboardsF    func(context.Context, platform.DashboardFilter) ([]*platform.Dashboard, int, error)
	UpdateDashboardF   func(context.Context, platform.ID, platform.DashboardUpdate) (*platform.Dashboard, error)
	DeleteDashboardF   func(context.Context, platform.ID) error
}

func (s *DashboardService) FindDashboardByID(ctx context.Context, id platform.ID) (*platform.Dashboard, error) {
	return s.FindDashboardByIDF(ctx, id)
}

func (s *DashboardService) FindDashboards(ctx context.Context, filter platform.DashboardFilter) ([]*platform.Dashboard, int, error) {
	return s.FindDashboardsF(ctx, filter)
}

func (s *DashboardService) CreateDashboard(ctx context.Context, b *platform.Dashboard) error {
	return s.CreateDashboardF(ctx, b)
}

func (s *DashboardService) UpdateDashboard(ctx context.Context, id platform.ID, upd platform.DashboardUpdate) (*platform.Dashboard, error) {
	return s.UpdateDashboardF(ctx, id, upd)
}

func (s *DashboardService) DeleteDashboard(ctx context.Context, id platform.ID) error {
	return s.DeleteDashboardF(ctx, id)
}
