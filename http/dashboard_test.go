package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/influxdata/platform"
	"github.com/influxdata/platform/mock"
	"github.com/julienschmidt/httprouter"
)

func TestService_handleGetDashboards(t *testing.T) {
	type fields struct {
		DashboardService platform.DashboardService
	}
	type args struct {
		queryParams map[string][]string
	}
	type wants struct {
		statusCode  int
		contentType string
		body        string
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		wants  wants
	}{
		{
			name: "get all dashboards",
			fields: fields{
				&mock.DashboardService{
					FindDashboardsF: func(ctx context.Context, filter platform.DashboardFilter) ([]*platform.Dashboard, int, error) {
						return []*platform.Dashboard{
							{
								ID:   platform.ID("0"),
								Name: "hello",
								Cells: []platform.DashboardCell{
									{
										X:   1,
										Y:   2,
										W:   3,
										H:   4,
										Ref: "/v2/cells/12",
									},
								},
							},
							{
								ID:   platform.ID("2"),
								Name: "example",
							},
						}, 2, nil
					},
				},
			},
			args: args{},
			wants: wants{
				statusCode:  http.StatusOK,
				contentType: "application/json; charset=utf-8",
				body: `
{
  "links": {
    "self": "/v2/dashboards"
  },
  "dashboards": [
    {
      "id": "30",
      "name": "hello",
      "cells": [
        {
          "x": 1,
          "y": 2,
          "w": 3,
          "h": 4,
          "ref": "/v2/cells/12"
        }
      ],
      "links": {
        "self": "/v2/dashboards/30"
      }
    },
    {
      "id": "32",
      "name": "example",
      "cells": [],
      "links": {
        "self": "/v2/dashboards/32"
      }
    }
  ]
}
`,
			},
		},
		{
			name: "get all dashboards when there are none",
			fields: fields{
				&mock.DashboardService{
					FindDashboardsF: func(ctx context.Context, filter platform.DashboardFilter) ([]*platform.Dashboard, int, error) {
						return []*platform.Dashboard{}, 0, nil
					},
				},
			},
			args: args{},
			wants: wants{
				statusCode:  http.StatusOK,
				contentType: "application/json; charset=utf-8",
				body: `
{
  "links": {
    "self": "/v2/dashboards"
  },
  "dashboards": []
}`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewDashboardHandler()
			h.DashboardService = tt.fields.DashboardService

			r := httptest.NewRequest("GET", "http://any.url", nil)

			qp := r.URL.Query()
			for k, vs := range tt.args.queryParams {
				for _, v := range vs {
					qp.Add(k, v)
				}
			}
			r.URL.RawQuery = qp.Encode()

			w := httptest.NewRecorder()

			h.handleGetDashboards(w, r)

			res := w.Result()
			content := res.Header.Get("Content-Type")
			body, _ := ioutil.ReadAll(res.Body)

			if res.StatusCode != tt.wants.statusCode {
				t.Errorf("%q. handleGetDashboards() = %v, want %v", tt.name, res.StatusCode, tt.wants.statusCode)
			}
			if tt.wants.contentType != "" && content != tt.wants.contentType {
				t.Errorf("%q. handleGetDashboards() = %v, want %v", tt.name, content, tt.wants.contentType)
			}
			if eq, _ := jsonEqual(string(body), tt.wants.body); tt.wants.body != "" && !eq {
				t.Errorf("%q. handleGetDashboards() = \n***%v***\n,\nwant\n***%v***", tt.name, string(body), tt.wants.body)
			}

		})
	}
}

func TestService_handleGetDashboard(t *testing.T) {
	type fields struct {
		DashboardService platform.DashboardService
	}
	type args struct {
		id string
	}
	type wants struct {
		statusCode  int
		contentType string
		body        string
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		wants  wants
	}{
		{
			name: "get a dashboard by id",
			fields: fields{
				&mock.DashboardService{
					FindDashboardByIDF: func(ctx context.Context, id platform.ID) (*platform.Dashboard, error) {
						if bytes.Equal(id, mustParseID("020f755c3c082000")) {
							return &platform.Dashboard{
								ID:   mustParseID("020f755c3c082000"),
								Name: "hello",
								Cells: []platform.DashboardCell{
									{
										X:   1,
										Y:   2,
										W:   3,
										H:   4,
										Ref: "/v2/cells/12",
									},
								},
							}, nil
						}

						return nil, fmt.Errorf("not found")
					},
				},
			},
			args: args{
				id: "020f755c3c082000",
			},
			wants: wants{
				statusCode:  http.StatusOK,
				contentType: "application/json; charset=utf-8",
				body: `
{
  "id": "020f755c3c082000",
  "name": "hello",
  "cells": [
    {
      "x": 1,
      "y": 2,
      "w": 3,
      "h": 4,
      "ref": "/v2/cells/12"
    }
  ],
  "links": {
    "self": "/v2/dashboards/020f755c3c082000"
  }
}
`,
			},
		},
		{
			name: "not found",
			fields: fields{
				&mock.DashboardService{
					FindDashboardByIDF: func(ctx context.Context, id platform.ID) (*platform.Dashboard, error) {
						return nil, platform.ErrDashboardNotFound
					},
				},
			},
			args: args{
				id: "020f755c3c082000",
			},
			wants: wants{
				statusCode: http.StatusNotFound,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewDashboardHandler()
			h.DashboardService = tt.fields.DashboardService

			r := httptest.NewRequest("GET", "http://any.url", nil)

			r = r.WithContext(context.WithValue(
				context.TODO(),
				httprouter.ParamsKey,
				httprouter.Params{
					{
						Key:   "id",
						Value: tt.args.id,
					},
				}))

			w := httptest.NewRecorder()

			h.handleGetDashboard(w, r)

			res := w.Result()
			content := res.Header.Get("Content-Type")
			body, _ := ioutil.ReadAll(res.Body)

			if res.StatusCode != tt.wants.statusCode {
				t.Errorf("%q. handleGetDashboard() = %v, want %v", tt.name, res.StatusCode, tt.wants.statusCode)
			}
			if tt.wants.contentType != "" && content != tt.wants.contentType {
				t.Errorf("%q. handleGetDashboard() = %v, want %v", tt.name, content, tt.wants.contentType)
			}
			if eq, _ := jsonEqual(string(body), tt.wants.body); tt.wants.body != "" && !eq {
				t.Errorf("%q. handleGetDashboard() = \n***%v***\n,\nwant\n***%v***", tt.name, string(body), tt.wants.body)
			}
		})
	}
}

func TestService_handlePostDashboards(t *testing.T) {
	type fields struct {
		DashboardService platform.DashboardService
	}
	type args struct {
		dashboard *platform.Dashboard
	}
	type wants struct {
		statusCode  int
		contentType string
		body        string
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		wants  wants
	}{
		{
			name: "create a new dashboard",
			fields: fields{
				&mock.DashboardService{
					CreateDashboardF: func(ctx context.Context, c *platform.Dashboard) error {
						c.ID = mustParseID("020f755c3c082000")
						return nil
					},
				},
			},
			args: args{
				dashboard: &platform.Dashboard{
					Name: "hello",
					Cells: []platform.DashboardCell{
						{
							X:   1,
							Y:   2,
							W:   3,
							H:   4,
							Ref: "/v2/cells/12",
						},
					},
				},
			},
			wants: wants{
				statusCode:  http.StatusCreated,
				contentType: "application/json; charset=utf-8",
				body: `
{
  "id": "020f755c3c082000",
  "name": "hello",
  "cells": [
    {
      "x": 1,
      "y": 2,
      "w": 3,
      "h": 4,
      "ref": "/v2/cells/12"
    }
  ],
  "links": {
    "self": "/v2/dashboards/020f755c3c082000"
  }
}
`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewDashboardHandler()
			h.DashboardService = tt.fields.DashboardService

			b, err := json.Marshal(tt.args.dashboard)
			if err != nil {
				t.Fatalf("failed to unmarshal dashboard: %v", err)
			}

			r := httptest.NewRequest("GET", "http://any.url", bytes.NewReader(b))
			w := httptest.NewRecorder()

			h.handlePostDashboards(w, r)

			res := w.Result()
			content := res.Header.Get("Content-Type")
			body, _ := ioutil.ReadAll(res.Body)

			if res.StatusCode != tt.wants.statusCode {
				t.Errorf("%q. handlePostDashboard() = %v, want %v", tt.name, res.StatusCode, tt.wants.statusCode)
			}
			if tt.wants.contentType != "" && content != tt.wants.contentType {
				t.Errorf("%q. handlePostDashboard() = %v, want %v", tt.name, content, tt.wants.contentType)
			}
			if eq, _ := jsonEqual(string(body), tt.wants.body); tt.wants.body != "" && !eq {
				t.Errorf("%q. handlePostDashboard() = \n***%v***\n,\nwant\n***%v***", tt.name, string(body), tt.wants.body)
			}
		})
	}
}

func TestService_handleDeleteDashboard(t *testing.T) {
	type fields struct {
		DashboardService platform.DashboardService
	}
	type args struct {
		id string
	}
	type wants struct {
		statusCode  int
		contentType string
		body        string
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		wants  wants
	}{
		{
			name: "remove a dashboard by id",
			fields: fields{
				&mock.DashboardService{
					DeleteDashboardF: func(ctx context.Context, id platform.ID) error {
						if bytes.Equal(id, mustParseID("020f755c3c082000")) {
							return nil
						}

						return fmt.Errorf("wrong id")
					},
				},
			},
			args: args{
				id: "020f755c3c082000",
			},
			wants: wants{
				statusCode: http.StatusNoContent,
			},
		},
		{
			name: "dashboard not found",
			fields: fields{
				&mock.DashboardService{
					DeleteDashboardF: func(ctx context.Context, id platform.ID) error {
						return platform.ErrDashboardNotFound
					},
				},
			},
			args: args{
				id: "020f755c3c082000",
			},
			wants: wants{
				statusCode: http.StatusNotFound,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewDashboardHandler()
			h.DashboardService = tt.fields.DashboardService

			r := httptest.NewRequest("GET", "http://any.url", nil)

			r = r.WithContext(context.WithValue(
				context.TODO(),
				httprouter.ParamsKey,
				httprouter.Params{
					{
						Key:   "id",
						Value: tt.args.id,
					},
				}))

			w := httptest.NewRecorder()

			h.handleDeleteDashboard(w, r)

			res := w.Result()
			content := res.Header.Get("Content-Type")
			body, _ := ioutil.ReadAll(res.Body)

			if res.StatusCode != tt.wants.statusCode {
				t.Errorf("%q. handleDeleteDashboard() = %v, want %v", tt.name, res.StatusCode, tt.wants.statusCode)
			}
			if tt.wants.contentType != "" && content != tt.wants.contentType {
				t.Errorf("%q. handleDeleteDashboard() = %v, want %v", tt.name, content, tt.wants.contentType)
			}
			if eq, _ := jsonEqual(string(body), tt.wants.body); tt.wants.body != "" && !eq {
				t.Errorf("%q. handleDeleteDashboard() = \n***%v***\n,\nwant\n***%v***", tt.name, string(body), tt.wants.body)
			}
		})
	}
}

func TestService_handlePatchDashboard(t *testing.T) {
	type fields struct {
		DashboardService platform.DashboardService
	}
	type args struct {
		id    string
		name  string
		cells []platform.DashboardCell
	}
	type wants struct {
		statusCode  int
		contentType string
		body        string
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		wants  wants
	}{
		{
			name: "update a dashboard name",
			fields: fields{
				&mock.DashboardService{
					UpdateDashboardF: func(ctx context.Context, id platform.ID, upd platform.DashboardUpdate) (*platform.Dashboard, error) {
						if bytes.Equal(id, mustParseID("020f755c3c082000")) {
							d := &platform.Dashboard{
								ID:   mustParseID("020f755c3c082000"),
								Name: "hello",
								Cells: []platform.DashboardCell{
									{
										X:   1,
										Y:   2,
										W:   3,
										H:   4,
										Ref: "/v2/cells/12",
									},
								},
							}

							if upd.Name != nil {
								d.Name = *upd.Name
							}

							if upd.Cells != nil {
								d.Cells = upd.Cells
							}

							return d, nil
						}

						return nil, fmt.Errorf("not found")
					},
				},
			},
			args: args{
				id:   "020f755c3c082000",
				name: "example",
			},
			wants: wants{
				statusCode:  http.StatusOK,
				contentType: "application/json; charset=utf-8",
				body: `
{
  "id": "020f755c3c082000",
  "name": "example",
  "cells": [
    {
      "x": 1,
      "y": 2,
      "w": 3,
      "h": 4,
      "ref": "/v2/cells/12"
    }
  ],
  "links": {
    "self": "/v2/dashboards/020f755c3c082000"
  }
}
`,
			},
		},
		{
			name: "update a dashboard cells",
			fields: fields{
				&mock.DashboardService{
					UpdateDashboardF: func(ctx context.Context, id platform.ID, upd platform.DashboardUpdate) (*platform.Dashboard, error) {
						if bytes.Equal(id, mustParseID("020f755c3c082000")) {
							d := &platform.Dashboard{
								ID:   mustParseID("020f755c3c082000"),
								Name: "hello",
								Cells: []platform.DashboardCell{
									{
										X:   1,
										Y:   2,
										W:   3,
										H:   4,
										Ref: "/v2/cells/12",
									},
								},
							}

							if upd.Name != nil {
								d.Name = *upd.Name
							}

							if upd.Cells != nil {
								d.Cells = upd.Cells
							}

							return d, nil
						}

						return nil, fmt.Errorf("not found")
					},
				},
			},
			args: args{
				id: "020f755c3c082000",
				cells: []platform.DashboardCell{
					{
						X:   1,
						Y:   2,
						W:   3,
						H:   4,
						Ref: "/v2/cells/12",
					},
					{
						X:   2,
						Y:   3,
						W:   4,
						H:   5,
						Ref: "/v2/cells/1",
					},
				},
			},
			wants: wants{
				statusCode:  http.StatusOK,
				contentType: "application/json; charset=utf-8",
				body: `
{
  "id": "020f755c3c082000",
  "name": "hello",
  "cells": [
    {
      "x": 1,
      "y": 2,
      "w": 3,
      "h": 4,
      "ref": "/v2/cells/12"
    },
    {
      "x": 2,
      "y": 3,
      "w": 4,
      "h": 5,
      "ref": "/v2/cells/1"
    }
  ],
  "links": {
    "self": "/v2/dashboards/020f755c3c082000"
  }
}
`,
			},
		},
		{
			name: "update a dashboard with empty request body",
			fields: fields{
				&mock.DashboardService{
					UpdateDashboardF: func(ctx context.Context, id platform.ID, upd platform.DashboardUpdate) (*platform.Dashboard, error) {
						return nil, fmt.Errorf("not found")
					},
				},
			},
			args: args{
				id: "020f755c3c082000",
			},
			wants: wants{
				statusCode: http.StatusBadRequest,
			},
		},
		{
			name: "dashboard not found",
			fields: fields{
				&mock.DashboardService{
					UpdateDashboardF: func(ctx context.Context, id platform.ID, upd platform.DashboardUpdate) (*platform.Dashboard, error) {
						return nil, platform.ErrDashboardNotFound
					},
				},
			},
			args: args{
				id:   "020f755c3c082000",
				name: "hello",
			},
			wants: wants{
				statusCode: http.StatusNotFound,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewDashboardHandler()
			h.DashboardService = tt.fields.DashboardService

			upd := platform.DashboardUpdate{}
			if tt.args.name != "" {
				upd.Name = &tt.args.name
			}
			if tt.args.cells != nil {
				upd.Cells = tt.args.cells
			}

			b, err := json.Marshal(upd)
			if err != nil {
				t.Fatalf("failed to unmarshal dashboard update: %v", err)
			}

			r := httptest.NewRequest("GET", "http://any.url", bytes.NewReader(b))

			r = r.WithContext(context.WithValue(
				context.TODO(),
				httprouter.ParamsKey,
				httprouter.Params{
					{
						Key:   "id",
						Value: tt.args.id,
					},
				}))

			w := httptest.NewRecorder()

			h.handlePatchDashboard(w, r)

			res := w.Result()
			content := res.Header.Get("Content-Type")
			body, _ := ioutil.ReadAll(res.Body)

			if res.StatusCode != tt.wants.statusCode {
				t.Errorf("%q. handlePatchDashboard() = %v, want %v", tt.name, res.StatusCode, tt.wants.statusCode)
			}
			if tt.wants.contentType != "" && content != tt.wants.contentType {
				t.Errorf("%q. handlePatchDashboard() = %v, want %v", tt.name, content, tt.wants.contentType)
			}
			if eq, _ := jsonEqual(string(body), tt.wants.body); tt.wants.body != "" && !eq {
				t.Errorf("%q. handlePatchDashboard() = \n***%v***\n,\nwant\n***%v***", tt.name, string(body), tt.wants.body)
			}
		})
	}
}
