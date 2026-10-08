package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/printdreams/cryptocash-ton-battery/internal/price"
)

type PriceHandler struct {
	Oracle *price.Oracle
}

func NewPriceHandler(o *price.Oracle) *PriceHandler {
	return &PriceHandler{Oracle: o}
}

// Price godoc
// @Summary      Current charge pricing
// @Description  Returns the live TON price and the resulting nano-per-charge used for charge computation
// @Tags         charges
// @Produce      json
// @Success      200  {object}  price.Quote
// @Failure      503  {object}  handler.ErrorResponse
// @Router       /price [get]
func (h *PriceHandler) Price(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if h.Oracle == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "price-oracle-not-configured"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	writeJSON(w, http.StatusOK, h.Oracle.Quote(ctx))
}
