package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/printdreams/cryptocash-ton-battery/internal/relayer"
	"github.com/printdreams/cryptocash-ton-battery/internal/ton"
)

type RelayerHandler struct {
	Relayer *relayer.Relayer
	Ton     *ton.Client
}

func NewRelayerHandler(r *relayer.Relayer, t *ton.Client) *RelayerHandler {
	return &RelayerHandler{Relayer: r, Ton: t}
}

// Status godoc
// @Summary      Relayer status
// @Description  Shows the relayer wallet address and its on-chain balance and seqno (reads TON testnet)
// @Tags         relayer
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      503  {object}  map[string]string
// @Router       /relayer/status [get]
func (h *RelayerHandler) Status(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if h.Relayer == nil || h.Ton == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "relayer-not-configured"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	info, err := h.Ton.GetWalletInformation(ctx, h.Relayer.Address())
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"address": h.Relayer.Address(),
			"online":  false,
			"error":   "chain-unreachable",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"address":     h.Relayer.Address(),
		"online":      true,
		"balanceNano": info.Balance,
		"seqno":       info.Seqno,
		"deployed":    info.Deployed,
	})
}
