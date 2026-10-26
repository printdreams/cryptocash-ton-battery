package handler

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/printdreams/cryptocash-ton-battery/internal/auth"
	"github.com/printdreams/cryptocash-ton-battery/internal/emulate"
	"github.com/printdreams/cryptocash-ton-battery/internal/ledger"
	"github.com/printdreams/cryptocash-ton-battery/internal/message"
	"github.com/printdreams/cryptocash-ton-battery/internal/policy"
	"github.com/printdreams/cryptocash-ton-battery/internal/price"
	"github.com/printdreams/cryptocash-ton-battery/internal/relayer"
	"github.com/printdreams/cryptocash-ton-battery/internal/user"
)

type WalletEmulateHandler struct {
	Emulator     *emulate.Client
	Ledger       *ledger.Store
	Users        *user.Store
	Sender       *relayer.Sender
	MsgCfg       message.Config
	PolCfg       policy.Config
	GasProbeNano int64
	Price        *price.Oracle
}

func NewWalletEmulateHandler(e *emulate.Client, l *ledger.Store, u *user.Store, s *relayer.Sender, msg message.Config, pol policy.Config, gasProbeNano int64, pr *price.Oracle) *WalletEmulateHandler {
	return &WalletEmulateHandler{Emulator: e, Ledger: l, Users: u, Sender: s, MsgCfg: msg, PolCfg: pol, GasProbeNano: gasProbeNano, Price: pr}
}

func relayExternalBOC(ctx context.Context, users *user.Store, sender *relayer.Sender, userID, boc string, gasNano int64) (string, error) {
	u, err := users.Get(ctx, userID)
	if err != nil {
		return "", errors.New("user-not-found")
	}
	pub, err := hex.DecodeString(u.PublicKey)
	if err != nil || len(pub) != 32 {
		return "", errors.New("bad-user-key")
	}
	body, err := base64.StdEncoding.DecodeString(boc)
	if err != nil {
		return "", errors.New("malformed-message")
	}
	return sender.BuildRelayExternalBOC(ctx, pub, body, gasNano)
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
	var emulDetail string

	if _, perr := message.Check(req.Boc, h.MsgCfg, now); perr != nil {
		facts.StaticReason = perr.Error()
	} else if h.Emulator != nil && h.Sender != nil && h.Users != nil {
		emBOC, berr := relayExternalBOC(r.Context(), h.Users, h.Sender, userID, req.Boc, h.GasProbeNano)
		if berr != nil {
			emulDetail = berr.Error()
		} else {
			ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
			em, eerr := h.Emulator.EmulateTrace(ctx, emBOC, req.IgnoreSignature)
			cancel()
			if eerr != nil {
				emulDetail = eerr.Error()
			} else {
				facts.Emulated = true
				facts.EmulationSuccess = em.Success
				facts.TotalFeesNano = em.TotalFeesNano
				facts.OutMessages = em.OutMessages
				facts.Destinations = em.Destinations
			}
		}
	}

	if acc, aerr := h.Ledger.Balance(r.Context(), userID); aerr == nil {
		facts.AvailableCharges = acc.Available()
	}

	polCfg := h.PolCfg
	if h.Price != nil {
		if n := h.Price.NanoPerCharge(r.Context()); n > 0 {
			polCfg.NanoPerCharge = n
		}
	}

	d := policy.Evaluate(facts, polCfg)

	w.Header().Set("Supported-By-Battery", strconv.FormatBool(d.SupportedByBattery))
	w.Header().Set("Allowed-By-Battery", strconv.FormatBool(d.AllowedByBattery))
	if d.RejectReason != "" {
		w.Header().Set("Reject-Reason", d.RejectReason)
	}

	resp := map[string]interface{}{
		"supported":    d.SupportedByBattery,
		"allowed":      d.AllowedByBattery,
		"rejectReason": d.RejectReason,
		"charge":       d.Charge,
	}
	if d.RejectReason == "emulation-failed" && emulDetail != "" {
		resp["detail"] = emulDetail
		w.Header().Set("Reject-Detail", emulDetail)
	}

	writeJSON(w, http.StatusOK, resp)
}
