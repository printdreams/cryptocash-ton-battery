package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/printdreams/cryptocash-ton-battery/internal/relayer"
)

type RelayerDevHandler struct {
	Sender *relayer.Sender
}

func NewRelayerDevHandler(s *relayer.Sender) *RelayerDevHandler {
	return &RelayerDevHandler{Sender: s}
}

// Send godoc
// @Summary      [TO BE DELETED] Dev-only relayer send
// @Description  TEST ONLY — makes the relayer send TON to an address (deploys the relayer on first send, pays gas). Enabled only when DEV_SIGN=true. Response carries header X-Dev-Route: TO-BE-DELETED.
// @Tags         dev
// @Produce      json
// @Param        to       query     string  true   "destination address"
// @Param        amount   query     string  false  "amount in TON (default 0.05)"
// @Param        comment  query     string  false  "optional comment"
// @Success      200      {object}  handler.RelayerDevSendResponse
// @Failure      400      {object}  handler.ErrorResponse
// @Failure      502      {object}  handler.ErrorResponse
// @Failure      503      {object}  handler.ErrorResponse
// @Router       /relayer/dev-send [post]
func (h *RelayerDevHandler) Send(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Dev-Route", "TO-BE-DELETED")

	if h.Sender == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "relayer-sender-not-ready"})
		return
	}

	to := r.URL.Query().Get("to")
	if to == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing-to"})
		return
	}
	amount := r.URL.Query().Get("amount")
	if amount == "" {
		amount = "0.05"
	}
	comment := r.URL.Query().Get("comment")
	if comment == "" {
		comment = "battery-test"
	}

	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()

	hash, err := h.Sender.Send(ctx, to, amount, comment)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "send-failed", "detail": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"txHash":   hash,
		"from":     h.Sender.Address(),
		"to":       to,
		"amount":   amount,
		"explorer": "https://testnet.tonviewer.com/" + h.Sender.Address(),
	})
}
