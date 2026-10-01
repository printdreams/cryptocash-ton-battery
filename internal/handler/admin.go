package handler

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/printdreams/cryptocash-ton-battery/internal/ledger"
)

type AdminHandler struct {
	Ledger *ledger.Store
}

func NewAdminHandler(l *ledger.Store) *AdminHandler {
	return &AdminHandler{Ledger: l}
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
// @Success      200   {object}  map[string]interface{}
// @Failure      400   {object}  map[string]string
// @Failure      401   {object}  map[string]string
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

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"userId":     res.Account.UserID,
		"balance":    res.Account.Balance,
		"reserved":   res.Account.Reserved,
		"available":  res.Account.Available(),
		"idempotent": res.Idempotent,
		"entryId":    res.Entry.ID,
	})
}

// Balance godoc
// @Summary      Get a user's charges balance (admin)
// @Description  Returns balance, reserved and available charges for a user. Requires the admin token.
// @Tags         admin
// @Produce      json
// @Param        id  path      string  true  "user id"
// @Security     AdminToken
// @Success      200  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]string
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
