package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/printdreams/cryptocash-ton-battery/internal/emulate"
)

type TonapiHandler struct {
	Emulator *emulate.Client
}

func NewTonapiHandler(e *emulate.Client) *TonapiHandler {
	return &TonapiHandler{Emulator: e}
}

// Status godoc
// @Summary      tonapi key status
// @Description  Calls tonapi with the configured TON_API_KEY and reports whether the key is accepted (valid) or rejected (bogus). Does not reveal the key.
// @Tags         tonapi
// @Produce      json
// @Success      200  {object}  TonapiStatusResponse
// @Failure      503  {object}  map[string]string
// @Router       /tonapi/status [get]
func (h *TonapiHandler) Status(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if h.Emulator == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "emulator-not-configured"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	ks := h.Emulator.CheckKey(ctx)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"keyConfigured": ks.KeyConfigured,
		"valid":         ks.Valid,
		"withKeyStatus": ks.WithKeyStatus,
		"badKeyStatus":  ks.BadKeyStatus,
		"endpoint":      ks.Endpoint,
		"detail":        ks.Detail,
	})
}
