package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/printdreams/cryptocash-ton-battery/internal/auth"
	"github.com/printdreams/cryptocash-ton-battery/internal/emulate"
	"github.com/printdreams/cryptocash-ton-battery/internal/ledger"
	"github.com/printdreams/cryptocash-ton-battery/internal/message"
	"github.com/printdreams/cryptocash-ton-battery/internal/policy"
)

type WalletEmulateHandler struct {
	Emulator *emulate.Client
	Ledger   *ledger.Store
	MsgCfg   message.Config
	PolCfg   policy.Config
}

func NewWalletEmulateHandler(e *emulate.Client, l *ledger.Store, msg message.Config, pol policy.Config) *WalletEmulateHandler {
	return &WalletEmulateHandler{Emulator: e, Ledger: l, MsgCfg: msg, PolCfg: pol}
}

type walletEmulateRequest struct {
	Boc             string `json:"boc"`
	IgnoreSignature bool   `json:"ignoreSignature"`
}

// Emulate godoc
// @Summary      Emulate a message and return the battery decision
// @Description  Runs static checks, emulation and policy over a message BOC and returns Supported-By-Battery / Allowed-By-Battery / Reject-Reason plus the charge cost
// @Tags         battery
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      walletEmulateRequest  true  "message boc"
// @Success      200   {object}  map[string]interface{}
// @Failure      400   {object}  map[string]string
// @Failure      401   {object}  map[string]string
// @Router       /wallet/emulate [post]
func (h *WalletEmulateHandler) Emulate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID := auth.UserID(r.Context())
	if userID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req walletEmulateRequest
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64*1024))
	if err := dec.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "malformed-request"})
		return
	}
	if req.Boc == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing-boc"})
		return
	}

	now := time.Now().Unix()
	facts := policy.Facts{}

	if _, perr := message.Check(req.Boc, h.MsgCfg, now); perr != nil {
		facts.StaticReason = perr.Error()
	} else if h.Emulator != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
		em, eerr := h.Emulator.EmulateTrace(ctx, req.Boc, req.IgnoreSignature)
		cancel()
		if eerr == nil {
			facts.Emulated = true
			facts.EmulationSuccess = em.Success
			facts.TotalFeesNano = em.TotalFeesNano
			facts.OutMessages = em.OutMessages
			facts.Destinations = em.Destinations
		}
	}

	if acc, aerr := h.Ledger.Balance(r.Context(), userID); aerr == nil {
		facts.AvailableCharges = acc.Available()
	}

	d := policy.Evaluate(facts, h.PolCfg)

	w.Header().Set("Supported-By-Battery", strconv.FormatBool(d.SupportedByBattery))
	w.Header().Set("Allowed-By-Battery", strconv.FormatBool(d.AllowedByBattery))
	if d.RejectReason != "" {
		w.Header().Set("Reject-Reason", d.RejectReason)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"supported":    d.SupportedByBattery,
		"allowed":      d.AllowedByBattery,
		"rejectReason": d.RejectReason,
		"charge":       d.Charge,
	})
}
