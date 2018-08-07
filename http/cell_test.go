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

	"github.com/google/go-cmp/cmp"
	"github.com/influxdata/platform"
	"github.com/influxdata/platform/mock"
	"github.com/julienschmidt/httprouter"
)

func TestService_handleGetCells(t *testing.T) {
	type fields struct {
		CellService platform.CellService
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
			name: "get all cells",
			fields: fields{
				&mock.CellService{
					FindCellsF: func(ctx context.Context, filter platform.CellFilter) ([]*platform.Cell, int, error) {
						return []*platform.Cell{
							{
								CellContents: platform.CellContents{
									ID:   platform.ID("0"),
									Name: "hello",
								},
								Visualization: platform.V1Visualization{
									Type: "line",
								},
							},
							{
								CellContents: platform.CellContents{
									ID:   platform.ID("2"),
									Name: "example",
								},
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
    "self": "/v2/cells"
  },
  "cells": [
    {
      "id": "30",
      "name": "hello",
      "links": {
        "self": "/v2/cells/30"
      },
      "visualization": {
        "type": "chronograf-v1",
        "queries": null,
        "axes": null,
        "visualizationType": "line",
        "colors": null,
        "legend": {},
        "tableOptions": {
          "verticalTimeAxis": false,
          "sortBy": {
            "internalName": "",
            "displayName": "",
            "visible": false
          },
          "wrapping": "",
          "fixFirstColumn": false
        },
        "fieldOptions": null,
        "timeFormat": "",
        "decimalPlaces": {
          "isEnforced": false,
          "digits": 0
        }
      }
    },
    {
      "id": "32",
      "name": "example",
      "links": {
        "self": "/v2/cells/32"
      },
      "visualization": {
        "type": "empty"
      }
    }
  ]
}`,
			},
		},
		{
			name: "get all cells when there are none",
			fields: fields{
				&mock.CellService{
					FindCellsF: func(ctx context.Context, filter platform.CellFilter) ([]*platform.Cell, int, error) {
						return []*platform.Cell{}, 0, nil
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
    "self": "/v2/cells"
  },
  "cells": []
}`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewCellHandler()
			h.CellService = tt.fields.CellService

			r := httptest.NewRequest("GET", "http://any.url", nil)

			qp := r.URL.Query()
			for k, vs := range tt.args.queryParams {
				for _, v := range vs {
					qp.Add(k, v)
				}
			}
			r.URL.RawQuery = qp.Encode()

			w := httptest.NewRecorder()

			h.handleGetCells(w, r)

			res := w.Result()
			content := res.Header.Get("Content-Type")
			body, _ := ioutil.ReadAll(res.Body)

			if res.StatusCode != tt.wants.statusCode {
				t.Errorf("%q. handleGetCells() = %v, want %v", tt.name, res.StatusCode, tt.wants.statusCode)
			}
			if tt.wants.contentType != "" && content != tt.wants.contentType {
				t.Errorf("%q. handleGetCells() = %v, want %v", tt.name, content, tt.wants.contentType)
			}
			if eq, _ := jsonEqual(string(body), tt.wants.body); tt.wants.body != "" && !eq {
				t.Errorf("%q. handleGetCells() = \n***%v***\n,\nwant\n***%v***", tt.name, string(body), tt.wants.body)
			}

		})
	}
}

func TestService_handleGetCell(t *testing.T) {
	type fields struct {
		CellService platform.CellService
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
			name: "get a cell by id",
			fields: fields{
				&mock.CellService{
					FindCellByIDF: func(ctx context.Context, id platform.ID) (*platform.Cell, error) {
						return &platform.Cell{
							CellContents: platform.CellContents{
								ID:   mustParseID("020f755c3c082000"),
								Name: "example",
							},
						}, nil
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
  "name": "example",
  "links": {
    "self": "/v2/cells/020f755c3c082000"
  },
  "visualization": {
    "type": "empty"
  }
}
`,
			},
		},
		{
			name: "not found",
			fields: fields{
				&mock.CellService{
					FindCellByIDF: func(ctx context.Context, id platform.ID) (*platform.Cell, error) {
						return nil, platform.ErrCellNotFound
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
			h := NewCellHandler()
			h.CellService = tt.fields.CellService

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

			h.handleGetCell(w, r)

			res := w.Result()
			content := res.Header.Get("Content-Type")
			body, _ := ioutil.ReadAll(res.Body)

			if res.StatusCode != tt.wants.statusCode {
				t.Errorf("%q. handleGetCell() = %v, want %v", tt.name, res.StatusCode, tt.wants.statusCode)
			}
			if tt.wants.contentType != "" && content != tt.wants.contentType {
				t.Errorf("%q. handleGetCell() = %v, want %v", tt.name, content, tt.wants.contentType)
			}
			if eq, _ := jsonEqual(string(body), tt.wants.body); tt.wants.body != "" && !eq {
				t.Errorf("%q. handleGetCell() = \n***%v***\n,\nwant\n***%v***", tt.name, string(body), tt.wants.body)
			}
		})
	}
}

func TestService_handlePostCells(t *testing.T) {
	type fields struct {
		CellService platform.CellService
	}
	type args struct {
		cell *platform.Cell
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
			name: "create a new cell",
			fields: fields{
				&mock.CellService{
					CreateCellF: func(ctx context.Context, c *platform.Cell) error {
						c.ID = mustParseID("020f755c3c082000")
						return nil
					},
				},
			},
			args: args{
				cell: &platform.Cell{
					CellContents: platform.CellContents{
						Name: "hello",
					},
					Visualization: platform.V1Visualization{
						Type: "line",
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
  "links": {
    "self": "/v2/cells/020f755c3c082000"
  },
  "visualization": {
    "type": "chronograf-v1",
    "queries": null,
    "axes": null,
    "visualizationType": "line",
    "colors": null,
    "legend": {},
    "tableOptions": {
      "verticalTimeAxis": false,
      "sortBy": {
        "internalName": "",
        "displayName": "",
        "visible": false
      },
      "wrapping": "",
      "fixFirstColumn": false
    },
    "fieldOptions": null,
    "timeFormat": "",
    "decimalPlaces": {
      "isEnforced": false,
      "digits": 0
    }
  }
}
`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewCellHandler()
			h.CellService = tt.fields.CellService

			b, err := json.Marshal(tt.args.cell)
			if err != nil {
				t.Fatalf("failed to unmarshal cell: %v", err)
			}

			r := httptest.NewRequest("GET", "http://any.url", bytes.NewReader(b))
			w := httptest.NewRecorder()

			h.handlePostCells(w, r)

			res := w.Result()
			content := res.Header.Get("Content-Type")
			body, _ := ioutil.ReadAll(res.Body)

			if res.StatusCode != tt.wants.statusCode {
				t.Errorf("%q. handlePostCells() = %v, want %v", tt.name, res.StatusCode, tt.wants.statusCode)
			}
			if tt.wants.contentType != "" && content != tt.wants.contentType {
				t.Errorf("%q. handlePostCells() = %v, want %v", tt.name, content, tt.wants.contentType)
			}
			if eq, _ := jsonEqual(string(body), tt.wants.body); tt.wants.body != "" && !eq {
				t.Errorf("%q. handlePostCells() = \n***%v***\n,\nwant\n***%v***", tt.name, string(body), tt.wants.body)
			}
		})
	}
}

func TestService_handleDeleteCell(t *testing.T) {
	type fields struct {
		CellService platform.CellService
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
			name: "remove a cell by id",
			fields: fields{
				&mock.CellService{
					DeleteCellF: func(ctx context.Context, id platform.ID) error {
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
			name: "cell not found",
			fields: fields{
				&mock.CellService{
					DeleteCellF: func(ctx context.Context, id platform.ID) error {
						return platform.ErrCellNotFound
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
			h := NewCellHandler()
			h.CellService = tt.fields.CellService

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

			h.handleDeleteCell(w, r)

			res := w.Result()
			content := res.Header.Get("Content-Type")
			body, _ := ioutil.ReadAll(res.Body)

			if res.StatusCode != tt.wants.statusCode {
				t.Errorf("%q. handleDeleteCell() = %v, want %v", tt.name, res.StatusCode, tt.wants.statusCode)
			}
			if tt.wants.contentType != "" && content != tt.wants.contentType {
				t.Errorf("%q. handleDeleteCell() = %v, want %v", tt.name, content, tt.wants.contentType)
			}
			if eq, _ := jsonEqual(string(body), tt.wants.body); tt.wants.body != "" && !eq {
				t.Errorf("%q. handleDeleteCell() = \n***%v***\n,\nwant\n***%v***", tt.name, string(body), tt.wants.body)
			}
		})
	}
}

func TestService_handlePatchCell(t *testing.T) {
	type fields struct {
		CellService platform.CellService
	}
	type args struct {
		id            string
		name          string
		visualization platform.Visualization
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
			name: "update a cell",
			fields: fields{
				&mock.CellService{
					UpdateCellF: func(ctx context.Context, id platform.ID, upd platform.CellUpdate) (*platform.Cell, error) {
						if bytes.Equal(id, mustParseID("020f755c3c082000")) {
							return &platform.Cell{
								CellContents: platform.CellContents{
									ID:   mustParseID("020f755c3c082000"),
									Name: "example",
								},
								Visualization: platform.V1Visualization{
									Type: "line",
								},
							}, nil
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
  "links": {
    "self": "/v2/cells/020f755c3c082000"
  },
  "visualization": {
    "type": "chronograf-v1",
    "queries": null,
    "axes": null,
    "visualizationType": "line",
    "colors": null,
    "legend": {},
    "tableOptions": {
      "verticalTimeAxis": false,
      "sortBy": {
        "internalName": "",
        "displayName": "",
        "visible": false
      },
      "wrapping": "",
      "fixFirstColumn": false
    },
    "fieldOptions": null,
    "timeFormat": "",
    "decimalPlaces": {
      "isEnforced": false,
      "digits": 0
    }
  }
}
`,
			},
		},
		{
			name: "update a cell with empty request body",
			fields: fields{
				&mock.CellService{
					UpdateCellF: func(ctx context.Context, id platform.ID, upd platform.CellUpdate) (*platform.Cell, error) {
						if bytes.Equal(id, mustParseID("020f755c3c082000")) {
							return &platform.Cell{
								CellContents: platform.CellContents{
									ID:   mustParseID("020f755c3c082000"),
									Name: "example",
								},
								Visualization: platform.V1Visualization{
									Type: "line",
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
				statusCode: http.StatusBadRequest,
			},
		},
		{
			name: "cell not found",
			fields: fields{
				&mock.CellService{
					UpdateCellF: func(ctx context.Context, id platform.ID, upd platform.CellUpdate) (*platform.Cell, error) {
						return nil, platform.ErrCellNotFound
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
			h := NewCellHandler()
			h.CellService = tt.fields.CellService

			upd := platform.CellUpdate{}
			if tt.args.name != "" {
				upd.Name = &tt.args.name
			}
			if tt.args.visualization != nil {
				upd.Visualization = tt.args.visualization
			}

			b, err := json.Marshal(upd)
			if err != nil {
				t.Fatalf("failed to unmarshal cell update: %v", err)
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

			h.handlePatchCell(w, r)

			res := w.Result()
			content := res.Header.Get("Content-Type")
			body, _ := ioutil.ReadAll(res.Body)

			if res.StatusCode != tt.wants.statusCode {
				t.Errorf("%q. handlePatchCell() = %v, want %v", tt.name, res.StatusCode, tt.wants.statusCode)
			}
			if tt.wants.contentType != "" && content != tt.wants.contentType {
				t.Errorf("%q. handlePatchCell() = %v, want %v", tt.name, content, tt.wants.contentType)
			}
			if eq, _ := jsonEqual(string(body), tt.wants.body); tt.wants.body != "" && !eq {
				t.Errorf("%q. handlePatchCell() = \n***%v***\n,\nwant\n***%v***", tt.name, string(body), tt.wants.body)
			}
		})
	}
}

func jsonEqual(s1, s2 string) (eq bool, err error) {
	var o1, o2 interface{}

	if err = json.Unmarshal([]byte(s1), &o1); err != nil {
		return
	}
	if err = json.Unmarshal([]byte(s2), &o2); err != nil {
		return
	}

	return cmp.Equal(o1, o2), nil
}

func mustParseID(i string) platform.ID {
	id, err := platform.IDFromString(i)
	if err != nil {
		panic(err)
	}
	return *id
}
