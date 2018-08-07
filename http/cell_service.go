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

// CellHandler is the handler for the cell service
type CellHandler struct {
	*httprouter.Router

	CellService platform.CellService
}

// NewCellHandler returns a new instance of CellHandler.
func NewCellHandler() *CellHandler {
	h := &CellHandler{
		Router: httprouter.New(),
	}

	h.HandlerFunc("POST", "/v2/cells", h.handlePostCells)
	h.HandlerFunc("GET", "/v2/cells", h.handleGetCells)
	h.HandlerFunc("GET", "/v2/cells/:id", h.handleGetCell)
	h.HandlerFunc("DELETE", "/v2/cells/:id", h.handleDeleteCell)
	h.HandlerFunc("PATCH", "/v2/cells/:id", h.handlePatchCell)
	return h
}

type cellLinks struct {
	Self string `json:"self"`
}

type cellResponse struct {
	platform.Cell
	Links cellLinks `json:"links"`
}

func (r cellResponse) MarshalJSON() ([]byte, error) {
	vis, err := platform.MarshalVisualizationJSON(r.Visualization)
	if err != nil {
		return nil, err
	}

	return json.Marshal(struct {
		platform.CellContents
		Links         cellLinks       `json:"links"`
		Visualization json.RawMessage `json:"visualization"`
	}{
		CellContents:  r.CellContents,
		Links:         r.Links,
		Visualization: vis,
	})
}

func newCellResponse(c *platform.Cell) cellResponse {
	return cellResponse{
		Links: cellLinks{
			Self: fmt.Sprintf("/v2/cells/%s", c.ID),
		},
		Cell: *c,
	}
}

// handleGetCells returns all cells within the store.
func (h *CellHandler) handleGetCells(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// TODO(desa): support filtering via query params
	cells, _, err := h.CellService.FindCells(ctx, platform.CellFilter{})
	if err != nil {
		EncodeError(ctx, errors.InternalErrorf("Error loading cells: %v", err), w)
		return
	}

	if err := encodeResponse(ctx, w, http.StatusOK, newGetCellsResponse(cells)); err != nil {
		EncodeError(ctx, err, w)
		return
	}
}

type getCellsLinks struct {
	Self string `json:"self"`
}

type getCellsResponse struct {
	Links getCellsLinks  `json:"links"`
	Cells []cellResponse `json:"cells"`
}

func newGetCellsResponse(cells []*platform.Cell) getCellsResponse {
	res := getCellsResponse{
		Links: getCellsLinks{
			Self: "/v2/cells",
		},
		Cells: make([]cellResponse, 0, len(cells)),
	}

	for _, cell := range cells {
		res.Cells = append(res.Cells, newCellResponse(cell))
	}

	return res
}

// handlePostCells creates a new cell.
func (h *CellHandler) handlePostCells(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	req, err := decodePostCellRequest(ctx, r)
	if err != nil {
		EncodeError(ctx, err, w)
		return
	}
	if err := h.CellService.CreateCell(ctx, req.Cell); err != nil {
		EncodeError(ctx, errors.InternalErrorf("Error loading cells: %v", err), w)
		return
	}

	if err := encodeResponse(ctx, w, http.StatusCreated, newCellResponse(req.Cell)); err != nil {
		EncodeError(ctx, err, w)
		return
	}
}

type postCellRequest struct {
	Cell *platform.Cell
}

func decodePostCellRequest(ctx context.Context, r *http.Request) (*postCellRequest, error) {
	c := &platform.Cell{}
	if err := json.NewDecoder(r.Body).Decode(c); err != nil {
		return nil, err
	}
	return &postCellRequest{
		Cell: c,
	}, nil
}

// hanldeGetCell retrieves a cell by ID.
func (h *CellHandler) handleGetCell(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	req, err := decodeGetCellRequest(ctx, r)
	if err != nil {
		EncodeError(ctx, err, w)
		return
	}

	cell, err := h.CellService.FindCellByID(ctx, req.CellID)
	if err != nil {
		if err == platform.ErrCellNotFound {
			err = errors.New(err.Error(), errors.NotFound)
		}
		EncodeError(ctx, err, w)
		return
	}

	if err := encodeResponse(ctx, w, http.StatusOK, newCellResponse(cell)); err != nil {
		EncodeError(ctx, err, w)
		return
	}
}

type getCellRequest struct {
	CellID platform.ID
}

func decodeGetCellRequest(ctx context.Context, r *http.Request) (*getCellRequest, error) {
	params := httprouter.ParamsFromContext(ctx)
	id := params.ByName("id")
	if id == "" {
		return nil, errors.InvalidDataf("url missing id")
	}

	var i platform.ID
	if err := i.DecodeFromString(id); err != nil {
		return nil, err
	}

	return &getCellRequest{
		CellID: i,
	}, nil
}

// handleDeleteCell removes a cell by ID.
func (h *CellHandler) handleDeleteCell(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	req, err := decodeDeleteCellRequest(ctx, r)
	if err != nil {
		EncodeError(ctx, err, w)
		return
	}

	if err := h.CellService.DeleteCell(ctx, req.CellID); err != nil {
		if err == platform.ErrCellNotFound {
			err = errors.New(err.Error(), errors.NotFound)
		}
		EncodeError(ctx, err, w)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type deleteCellRequest struct {
	CellID platform.ID
}

func decodeDeleteCellRequest(ctx context.Context, r *http.Request) (*deleteCellRequest, error) {
	params := httprouter.ParamsFromContext(ctx)
	id := params.ByName("id")
	if id == "" {
		return nil, errors.InvalidDataf("url missing id")
	}

	var i platform.ID
	if err := i.DecodeFromString(id); err != nil {
		return nil, err
	}

	return &deleteCellRequest{
		CellID: i,
	}, nil
}

// handlePatchCell updates a cell.
func (h *CellHandler) handlePatchCell(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	req, err := decodePatchCellRequest(ctx, r)
	if err != nil {
		EncodeError(ctx, err, w)
		return
	}
	cell, err := h.CellService.UpdateCell(ctx, req.CellID, req.Upd)
	if err != nil {
		if err == platform.ErrCellNotFound {
			err = errors.New(err.Error(), errors.NotFound)
		}
		EncodeError(ctx, err, w)
		return
	}

	if err := encodeResponse(ctx, w, http.StatusOK, newCellResponse(cell)); err != nil {
		EncodeError(ctx, err, w)
		return
	}
}

type patchCellRequest struct {
	CellID platform.ID
	Upd    platform.CellUpdate
}

func decodePatchCellRequest(ctx context.Context, r *http.Request) (*patchCellRequest, error) {
	req := &patchCellRequest{}
	upd := platform.CellUpdate{}
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

	req.CellID = i

	if err := req.Valid(); err != nil {
		return nil, errors.MalformedDataf(err.Error())
	}

	return req, nil
}

// Valid validates that the cell ID is non zero valued and update has expected values set.
func (r *patchCellRequest) Valid() error {
	if len(r.CellID) == 0 {
		return fmt.Errorf("missing cell ID")
	}

	return r.Upd.Valid()
}
