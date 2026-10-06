package handler

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/printdreams/cryptocash-ton-battery/internal/auth"
	"github.com/printdreams/cryptocash-ton-battery/internal/emulate"
	"github.com/printdreams/cryptocash-ton-battery/internal/ledger"
	"github.com/printdreams/cryptocash-ton-battery/internal/message"
	"github.com/printdreams/cryptocash-ton-battery/internal/policy"
	"github.com/printdreams/cryptocash-ton-battery/internal/relayer"
	"github.com/printdreams/cryptocash-ton-battery/internal/user"
)

type MessageHandler struct {
	Emulator *emulate.Client
	Ledger   *ledger.Store
	Users    *user.Store
	Sender   *relayer.Sender
	MsgCfg   message.Config
	PolCfg   policy.Config
	GasTON   string
}

func NewMessageHandler(e *emulate.Client, l *ledger.Store, u *user.Store, s *relayer.Sender, msg message.Config, pol policy.Config, gasTON string) *MessageHandler {
	return &MessageHandler{Emulator: e, Ledger: l, Users: u, Sender: s, MsgCfg: msg, PolCfg: pol, GasTON: gasTON}
}

type sendRequest struct {
	Boc             string `json:"boc"`
	IgnoreSignature bool   `json:"ignoreSignature"`
}

// Send godoc
// @Summary      Send a gasless message
// @Description  Runs the battery decision, reserves charges, relays the user's signed message via the relayer and settles or releases the charges
// @Tags         battery
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      sendRequest  true  "message boc"
// @Success      200   {object}  map[string]interface{}
// @Failure      400   {object}  map[string]string
// @Failure      401   {object}  map[string]string
// @Failure      502   {object}  map[string]string
// @Router       /message [post]
func (h *MessageHandler) Send(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID := auth.UserID(r.Context())
	if userID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req sendRequest
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

	if !d.AllowedByBattery {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"allowed":      false,
			"supported":    d.SupportedByBattery,
			"rejectReason": d.RejectReason,
			"charge":       d.Charge,
		})
		return
	}

	if h.Sender == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "relayer-sender-not-ready"})
		return
	}

	u, uerr := h.Users.Get(r.Context(), userID)
	if uerr != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "user-not-found"})
		return
	}
	pubBytes, kerr := hex.DecodeString(u.PublicKey)
	if kerr != nil || len(pubBytes) != 32 {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "bad-user-key"})
		return
	}
	bocBytes, berr := base64.StdEncoding.DecodeString(req.Boc)
	if berr != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "malformed-message"})
		return
	}

	digest := sha256.Sum256(bocBytes)
	key := hex.EncodeToString(digest[:])

	if d.Charge > 0 {
		if _, rerr := h.Ledger.Reserve(r.Context(), userID, d.Charge, ledger.Ref{
			IdempotencyKey: "reserve:" + key,
			Type:           "message",
		}); rerr != nil {
			w.Header().Set("Allowed-By-Battery", "false")
			w.Header().Set("Reject-Reason", "insufficient-charges")
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"allowed":      false,
				"supported":    true,
				"rejectReason": "insufficient-charges",
				"charge":       d.Charge,
			})
			return
		}
	}

	sctx, scancel := context.WithTimeout(r.Context(), 90*time.Second)
	txHash, serr := h.Sender.RelayUserMessage(sctx, pubBytes, bocBytes, h.GasTON)
	scancel()

	if serr != nil {
		if d.Charge > 0 {
			_, _ = h.Ledger.Release(r.Context(), userID, d.Charge, ledger.Ref{
				IdempotencyKey: "release:" + key,
				Type:           "message",
			})
		}
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "send-failed", "detail": serr.Error()})
		return
	}

	if d.Charge > 0 {
		_, _ = h.Ledger.Settle(r.Context(), userID, d.Charge, ledger.Ref{
			IdempotencyKey: "settle:" + key,
			Type:           "message",
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status": "confirmed",
		"txHash": txHash,
		"charge": d.Charge,
	})
}
