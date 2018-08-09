package plan_test

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/influxdata/platform/query"
	"github.com/influxdata/platform/query/plan"
	"github.com/influxdata/platform/query/plan/plantest"
)

func TestBoundsIntersect(t *testing.T) {
	tests := []struct {
		name string
		a, b plan.BoundsSpec
		want plan.BoundsSpec
	}{
		{
			name: "contained",
			a: plan.BoundsSpec{
				Start: query.Time{
					IsRelative: true,
					Relative:   -1 * time.Hour,
				},
			},
			b: plan.BoundsSpec{
				Start: query.Time{
					IsRelative: true,
					Relative:   -30 * time.Minute,
				},
			},
			want: plan.BoundsSpec{
				Start: query.Time{
					IsRelative: true,
					Relative:   -30 * time.Minute,
				},
			},
		},
		{
			name: "contained sym",
			a: plan.BoundsSpec{
				Start: query.Time{
					IsRelative: true,
					Relative:   -1 * time.Hour,
				},
			},
			b: plan.BoundsSpec{
				Start: query.Time{
					IsRelative: true,
					Relative:   -30 * time.Minute,
				},
			},
			want: plan.BoundsSpec{
				Start: query.Time{
					IsRelative: true,
					Relative:   -30 * time.Minute,
				},
			},
		},
		{
			name: "no overlap",
			a: plan.BoundsSpec{
				Start: query.Time{
					IsRelative: true,
					Relative:   -1 * time.Hour,
				},
			},
			b: plan.BoundsSpec{
				Start: query.Time{
					IsRelative: true,
					Relative:   -3 * time.Hour,
				},
				Stop: query.Time{
					IsRelative: true,
					Relative:   -2 * time.Hour,
				},
			},
			want: plan.BoundsSpec{
				Start: query.Time{
					IsRelative: true,
					Relative:   -1 * time.Hour,
				},
			},
		},
		{
			name: "no overlap sym",
			a: plan.BoundsSpec{
				Start: query.Time{
					IsRelative: true,
					Relative:   -1 * time.Hour,
				},
			},
			b: plan.BoundsSpec{
				Start: query.Time{
					IsRelative: true,
					Relative:   -3 * time.Hour,
				},
				Stop: query.Time{
					IsRelative: true,
					Relative:   -2 * time.Hour,
				},
			},
			want: plan.BoundsSpec{
				Start: query.Time{
					IsRelative: true,
					Relative:   -1 * time.Hour,
				},
			},
		},
		{
			name: "overlap",
			a: plan.BoundsSpec{
				Start: query.Time{
					IsRelative: true,
					Relative:   -1 * time.Hour,
				},
			},
			b: plan.BoundsSpec{
				Start: query.Time{
					IsRelative: true,
					Relative:   -2 * time.Hour,
				},
				Stop: query.Time{
					IsRelative: true,
					Relative:   -30 * time.Minute,
				},
			},
			want: plan.BoundsSpec{
				Start: query.Time{
					IsRelative: true,
					Relative:   -1 * time.Hour,
				},
				Stop: query.Time{
					IsRelative: true,
					Relative:   -30 * time.Minute,
				},
			},
		},
		{
			name: "overlap sym",
			a: plan.BoundsSpec{
				Start: query.Time{
					IsRelative: true,
					Relative:   -1 * time.Hour,
				},
			},
			b: plan.BoundsSpec{
				Start: query.Time{
					IsRelative: true,
					Relative:   -2 * time.Hour,
				},
				Stop: query.Time{
					IsRelative: true,
					Relative:   -30 * time.Minute,
				},
			},
			want: plan.BoundsSpec{
				Start: query.Time{
					IsRelative: true,
					Relative:   -1 * time.Hour,
				},
				Stop: query.Time{
					IsRelative: true,
					Relative:   -30 * time.Minute,
				},
			},
		},
		{
			name: "both start zero",
			a: plan.BoundsSpec{
				Stop: query.Time{
					IsRelative: true,
					Relative:   -1 * time.Hour,
				},
			},
			b: plan.BoundsSpec{
				Stop: query.Time{
					IsRelative: true,
					Relative:   -20 * time.Minute,
				},
			},
			want: plan.BoundsSpec{
				Stop: query.Time{
					IsRelative: true,
					Relative:   -1 * time.Hour,
				},
			},
		},
		{
			name: "both start zero sym",
			a: plan.BoundsSpec{
				Stop: query.Time{
					IsRelative: true,
					Relative:   -1 * time.Hour,
				},
			},
			b: plan.BoundsSpec{
				Stop: query.Time{
					IsRelative: true,
					Relative:   -20 * time.Minute,
				},
			},
			want: plan.BoundsSpec{
				Stop: query.Time{
					IsRelative: true,
					Relative:   -1 * time.Hour,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.a.Intersect(tt.b, time.Now())
			if !cmp.Equal(got, tt.want) {
				t.Errorf("unexpected bounds -want/+got:\n%s", cmp.Diff(tt.want, got, plantest.CmpOptions...))
			}
		})
	}
}
