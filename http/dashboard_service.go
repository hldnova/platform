package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/influxdata/platform"
	"github.com/influxdata/platform/kit/errors"
	"github.com/julienschmidt/httprouter"
)

// DashboardHandler is the handler for the dashboard service
type DashboardHandler struct {
	*httprouter.Router

	DashboardService platform.DashboardService
}

// NewDashboardHandler returns a new instance of DashboardHandler.
func NewDashboardHandler() *DashboardHandler {
	h := &DashboardHandler{
		Router: httprouter.New(),
	}

	h.HandlerFunc("POST", "/v2/dashboards", h.handlePostDashboards)
	h.HandlerFunc("GET", "/v2/dashboards", h.handleGetDashboards)
	h.HandlerFunc("GET", "/v2/dashboards/:id", h.handleGetDashboard)
	h.HandlerFunc("DELETE", "/v2/dashboards/:id", h.handleDeleteDashboard)
	h.HandlerFunc("PATCH", "/v2/dashboards/:id", h.handlePatchDashboard)
	return h
}

type dashboardLinks struct {
	Self string `json:"self"`
}

type dashboardResponse struct {
	platform.Dashboard
	Links dashboardLinks `json:"links"`
}

func newDashboardResponse(d *platform.Dashboard) dashboardResponse {
	if d.Cells == nil {
		d.Cells = []platform.DashboardCell{}
	}
	return dashboardResponse{
		Links: dashboardLinks{
			Self: fmt.Sprintf("/v2/dashboards/%s", d.ID),
		},
		Dashboard: *d,
	}
}

// handleGetDashboards returns all dashboards within the store.
func (h *DashboardHandler) handleGetDashboards(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// TODO(desa): support filtering via query params
	dashboards, _, err := h.DashboardService.FindDashboards(ctx, platform.DashboardFilter{})
	if err != nil {
		EncodeError(ctx, errors.InternalErrorf("Error loading dashboards: %v", err), w)
		return
	}

	if err := encodeResponse(ctx, w, http.StatusOK, newGetDashboardsResponse(dashboards)); err != nil {
		EncodeError(ctx, err, w)
		return
	}
}

type getDashboardsLinks struct {
	Self string `json:"self"`
}

type getDashboardsResponse struct {
	Links      getDashboardsLinks  `json:"links"`
	Dashboards []dashboardResponse `json:"dashboards"`
}

func newGetDashboardsResponse(dashboards []*platform.Dashboard) getDashboardsResponse {
	res := getDashboardsResponse{
		Links: getDashboardsLinks{
			Self: "/v2/dashboards",
		},
		Dashboards: make([]dashboardResponse, 0, len(dashboards)),
	}

	for _, dashboard := range dashboards {
		res.Dashboards = append(res.Dashboards, newDashboardResponse(dashboard))
	}

	return res
}

// handlePostDashboards creates a new dashboard.
func (h *DashboardHandler) handlePostDashboards(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	req, err := decodePostDashboardRequest(ctx, r)
	if err != nil {
		EncodeError(ctx, err, w)
		return
	}
	if err := h.DashboardService.CreateDashboard(ctx, req.Dashboard); err != nil {
		EncodeError(ctx, errors.InternalErrorf("Error loading dashboards: %v", err), w)
		return
	}

	if err := encodeResponse(ctx, w, http.StatusCreated, newDashboardResponse(req.Dashboard)); err != nil {
		EncodeError(ctx, err, w)
		return
	}
}

type postDashboardRequest struct {
	Dashboard *platform.Dashboard
}

func decodePostDashboardRequest(ctx context.Context, r *http.Request) (*postDashboardRequest, error) {
	c := &platform.Dashboard{}
	if err := json.NewDecoder(r.Body).Decode(c); err != nil {
		return nil, err
	}
	return &postDashboardRequest{
		Dashboard: c,
	}, nil
}

// hanldeGetDashboard retrieves a dashboard by ID.
func (h *DashboardHandler) handleGetDashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	req, err := decodeGetDashboardRequest(ctx, r)
	if err != nil {
		EncodeError(ctx, err, w)
		return
	}

	dashboard, err := h.DashboardService.FindDashboardByID(ctx, req.DashboardID)
	if err != nil {
		if err == platform.ErrDashboardNotFound {
			err = errors.New(err.Error(), errors.NotFound)
		}
		EncodeError(ctx, err, w)
		return
	}

	if err := encodeResponse(ctx, w, http.StatusOK, newDashboardResponse(dashboard)); err != nil {
		EncodeError(ctx, err, w)
		return
	}
}

type getDashboardRequest struct {
	DashboardID platform.ID
}

func decodeGetDashboardRequest(ctx context.Context, r *http.Request) (*getDashboardRequest, error) {
	params := httprouter.ParamsFromContext(ctx)
	id := params.ByName("id")
	if id == "" {
		return nil, errors.InvalidDataf("url missing id")
	}

	var i platform.ID
	if err := i.DecodeFromString(id); err != nil {
		return nil, err
	}

	return &getDashboardRequest{
		DashboardID: i,
	}, nil
}

// handleDeleteDashboard removes a dashboard by ID.
func (h *DashboardHandler) handleDeleteDashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	req, err := decodeDeleteDashboardRequest(ctx, r)
	if err != nil {
		EncodeError(ctx, err, w)
		return
	}

	if err := h.DashboardService.DeleteDashboard(ctx, req.DashboardID); err != nil {
		if err == platform.ErrDashboardNotFound {
			err = errors.New(err.Error(), errors.NotFound)
		}
		EncodeError(ctx, err, w)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type deleteDashboardRequest struct {
	DashboardID platform.ID
}

func decodeDeleteDashboardRequest(ctx context.Context, r *http.Request) (*deleteDashboardRequest, error) {
	params := httprouter.ParamsFromContext(ctx)
	id := params.ByName("id")
	if id == "" {
		return nil, errors.InvalidDataf("url missing id")
	}

	var i platform.ID
	if err := i.DecodeFromString(id); err != nil {
		return nil, err
	}

	return &deleteDashboardRequest{
		DashboardID: i,
	}, nil
}

// handlePatchDashboard updates a dashboard.
func (h *DashboardHandler) handlePatchDashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	req, err := decodePatchDashboardRequest(ctx, r)
	if err != nil {
		EncodeError(ctx, err, w)
		return
	}
	dashboard, err := h.DashboardService.UpdateDashboard(ctx, req.DashboardID, req.Upd)
	if err != nil {
		if err == platform.ErrDashboardNotFound {
			err = errors.New(err.Error(), errors.NotFound)
		}
		EncodeError(ctx, err, w)
		return
	}

	if err := encodeResponse(ctx, w, http.StatusOK, newDashboardResponse(dashboard)); err != nil {
		EncodeError(ctx, err, w)
		return
	}
}

type patchDashboardRequest struct {
	DashboardID platform.ID
	Upd         platform.DashboardUpdate
}

func decodePatchDashboardRequest(ctx context.Context, r *http.Request) (*patchDashboardRequest, error) {
	req := &patchDashboardRequest{}
	upd := platform.DashboardUpdate{}
	if err := json.NewDecoder(r.Body).Decode(&upd); err != nil {
		return nil, errors.MalformedDataf(err.Error())
	}

	req.Upd = upd

	params := httprouter.ParamsFromContext(ctx)
	id := params.ByName("id")
	if id == "" {
		return nil, errors.InvalidDataf("url missing id")
	}
	var i platform.ID
	if err := i.DecodeFromString(id); err != nil {
		return nil, err
	}

	req.DashboardID = i

	if err := req.Valid(); err != nil {
		return nil, errors.MalformedDataf(err.Error())
	}

	return req, nil
}

// Valid validates that the dashboard ID is non zero valued and update has expected values set.
func (r *patchDashboardRequest) Valid() error {
	if len(r.DashboardID) == 0 {
		return fmt.Errorf("missing dashboard ID")
	}

	return r.Upd.Valid()
}
