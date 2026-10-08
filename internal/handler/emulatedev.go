package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/printdreams/cryptocash-ton-battery/internal/emulate"
)

type EmulateDevHandler struct {
	Emulator *emulate.Client
}

func NewEmulateDevHandler(e *emulate.Client) *EmulateDevHandler {
	return &EmulateDevHandler{Emulator: e}
}

type emulateRequest struct {
	Boc             string `json:"boc"`
	IgnoreSignature bool   `json:"ignoreSignature"`
}

// Emulate godoc
// @Summary      [TO BE DELETED] Dev-only message emulation
// @Description  TEST ONLY — emulates a message BOC via tonapi and returns success, total fees and destinations. Enabled only when DEV_SIGN=true.
// @Tags         dev
// @Accept       json
// @Produce      json
// @Param        body  body      emulateRequest  true  "message boc"
// @Success      200   {object}  handler.EmulateResponse
// @Failure      400   {object}  handler.ErrorResponse
// @Failure      502   {object}  handler.ErrorResponse
// @Router       /emulate [post]
func (h *EmulateDevHandler) Emulate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Dev-Route", "TO-BE-DELETED")

	if h.Emulator == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "emulator-not-configured"})
		return
	}

	var req emulateRequest
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64*1024))
	if err := dec.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "malformed-request"})
		return
	}
	if req.Boc == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing-boc"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
	defer cancel()

	res, err := h.Emulator.EmulateTrace(ctx, req.Boc, req.IgnoreSignature)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "emulate-failed", "detail": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":       res.Success,
		"totalFeesNano": res.TotalFeesNano,
		"outMessages":   res.OutMessages,
		"destinations":  res.Destinations,
	})
}
