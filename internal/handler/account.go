package handler

import (
	"net/http"

	"github.com/printdreams/cryptocash-ton-battery/internal/auth"
	"github.com/printdreams/cryptocash-ton-battery/internal/ledger"
	"github.com/printdreams/cryptocash-ton-battery/internal/user"
)

type AccountHandler struct {
	Users  *user.Store
	Ledger *ledger.Store
}

func NewAccountHandler(users *user.Store, l *ledger.Store) *AccountHandler {
	return &AccountHandler{Users: users, Ledger: l}
}

// Me godoc
// @Summary      Current user
// @Description  Returns the authenticated user's id and public key
// @Tags         auth
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]string
// @Router       /auth/me [get]
func (h *AccountHandler) Me(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID := auth.UserID(r.Context())
	if userID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	u, err := h.Users.Get(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{"userId": userID})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"userId":    userID,
		"publicKey": u.PublicKey,
	})
}

// Balance godoc
// @Summary      Charges balance
// @Description  Returns the authenticated user's balance, reserved and available charges
// @Tags         charges
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]string
// @Router       /balance [get]
func (h *AccountHandler) Balance(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID := auth.UserID(r.Context())
	if userID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
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

// Transactions godoc
// @Summary      Charges history
// @Description  Returns the authenticated user's ledger entries, newest first
// @Tags         charges
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]string
// @Router       /transactions [get]
func (h *AccountHandler) Transactions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID := auth.UserID(r.Context())
	if userID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	entries, err := h.Ledger.History(r.Context(), userID, 50)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "ledger-failed"})
		return
	}
	items := make([]map[string]interface{}, 0, len(entries))
	for _, e := range entries {
		items = append(items, map[string]interface{}{
			"id":            e.ID,
			"op":            e.Op,
			"amount":        e.Amount,
			"balanceAfter":  e.BalanceAfter,
			"reservedAfter": e.ReservedAfter,
			"reason":        e.Reason,
			"createdAt":     e.CreatedAt,
		})
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"userId":       userID,
		"transactions": items,
	})
}
