package handler

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/printdreams/cryptocash-ton-battery/internal/abuse"
	"github.com/printdreams/cryptocash-ton-battery/internal/audit"
	"github.com/printdreams/cryptocash-ton-battery/internal/ledger"
)

type AdminHandler struct {
	Ledger     *ledger.Store
	Killswitch *abuse.Killswitch
	Audit      *audit.Store
}

func NewAdminHandler(l *ledger.Store, ks *abuse.Killswitch, au *audit.Store) *AdminHandler {
	return &AdminHandler{Ledger: l, Killswitch: ks, Audit: au}
}

type creditRequest struct {
	Amount         int64  `json:"amount"`
	IdempotencyKey string `json:"idempotencyKey"`
	Reason         string `json:"reason"`
}

// Credit godoc
// @Summary      Credit charges to a user (admin)
// @Description  Mock top-up: adds charges to a user's balance. Requires the admin token. Idempotent per idempotencyKey.
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id    path      string         true  "user id"
// @Param        body  body      creditRequest  true  "amount + idempotency key"
// @Security     AdminToken
// @Success      200   {object}  handler.AdminCreditResponse
// @Failure      400   {object}  handler.ErrorResponse
// @Failure      401   {object}  handler.ErrorResponse
// @Router       /admin/users/{id}/credit [post]
func (h *AdminHandler) Credit(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID := chi.URLParam(r, "id")
	if userID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "malformed-request"})
		return
	}

	var req creditRequest
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8*1024))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "malformed-request"})
		return
	}
	if req.IdempotencyKey == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing-idempotency-key"})
		return
	}

	res, err := h.Ledger.Credit(r.Context(), userID, req.Amount, ledger.Ref{
		IdempotencyKey: req.IdempotencyKey,
		Type:           "admin",
		Reason:         req.Reason,
	})
	if err != nil {
		switch {
		case errors.Is(err, ledger.ErrInvalidAmount):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid-amount"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "ledger-failed"})
		}
		return
	}

	h.Audit.Append(r.Context(), "credit", "admin", map[string]interface{}{
		"userId":         userID,
		"amount":         req.Amount,
		"idempotencyKey": req.IdempotencyKey,
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"userId":     res.Account.UserID,
		"balance":    res.Account.Balance,
		"reserved":   res.Account.Reserved,
		"available":  res.Account.Available(),
		"idempotent": res.Idempotent,
		"entryId":    res.Entry.ID,
	})
}

type killswitchRequest struct {
	Paused bool `json:"paused"`
}

// SetKillswitch godoc
// @Summary      Set the battery kill switch (admin)
// @Description  Pauses or resumes all sends and prints. Requires the admin token.
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        body  body      killswitchRequest  true  "paused flag"
// @Security     AdminToken
// @Success      200   {object}  handler.KillswitchResponse
// @Failure      400   {object}  handler.ErrorResponse
// @Router       /admin/killswitch [post]
func (h *AdminHandler) SetKillswitch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req killswitchRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4*1024)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "malformed-request"})
		return
	}
	if h.Killswitch == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "killswitch-not-configured"})
		return
	}
	if err := h.Killswitch.SetPaused(r.Context(), req.Paused); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "killswitch-failed"})
		return
	}

	h.Audit.Append(r.Context(), "killswitch", "admin", map[string]interface{}{"paused": req.Paused})

	writeJSON(w, http.StatusOK, map[string]interface{}{"paused": req.Paused})
}

// GetKillswitch godoc
// @Summary      Get the battery kill switch (admin)
// @Description  Returns whether the battery is paused. Requires the admin token.
// @Tags         admin
// @Produce      json
// @Security     AdminToken
// @Success      200  {object}  handler.KillswitchResponse
// @Router       /admin/killswitch [get]
func (h *AdminHandler) GetKillswitch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	paused := false
	if h.Killswitch != nil {
		paused = h.Killswitch.Paused(r.Context())
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"paused": paused})
}

// Reconcile godoc
// @Summary      Reconcile a user's ledger (admin)
// @Description  Recomputes balance and reserved from the ledger entries and compares to the account. Requires the admin token.
// @Tags         admin
// @Produce      json
// @Param        id  path      string  true  "user id"
// @Security     AdminToken
// @Success      200  {object}  handler.ReconcileResponse
// @Failure      401  {object}  handler.ErrorResponse
// @Router       /admin/users/{id}/reconcile [get]
func (h *AdminHandler) Reconcile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID := chi.URLParam(r, "id")
	rep, err := h.Ledger.Reconcile(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "ledger-failed"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"userId":           rep.UserID,
		"accountBalance":   rep.AccountBalance,
		"accountReserved":  rep.AccountReserved,
		"computedBalance":  rep.ComputedBalance,
		"computedReserved": rep.ComputedReserved,
		"consistent":       rep.Consistent,
	})
}

// Balance godoc
// @Summary      Get a user's charges balance (admin)
// @Description  Returns balance, reserved and available charges for a user. Requires the admin token.
// @Tags         admin
// @Produce      json
// @Param        id  path      string  true  "user id"
// @Security     AdminToken
// @Success      200  {object}  handler.BalanceResponse
// @Failure      401  {object}  handler.ErrorResponse
// @Router       /admin/users/{id} [get]
func (h *AdminHandler) Balance(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID := chi.URLParam(r, "id")
	if userID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "malformed-request"})
		return
	}

	acc, err := h.Ledger.Balance(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "ledger-failed"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"userId":    acc.UserID,
		"balance":   acc.Balance,
		"reserved":  acc.Reserved,
		"available": acc.Available(),
	})
}

func AdminAuth(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			got := r.Header.Get("X-Admin-Token")
			if token == "" || subtle.ConstantTimeCompare([]byte(got), []byte(token)) != 1 {
				w.Header().Set("Content-Type", "application/json")
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
