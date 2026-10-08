package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/printdreams/cryptocash-ton-battery/internal/auth"
	"github.com/printdreams/cryptocash-ton-battery/internal/nonce"
	"github.com/printdreams/cryptocash-ton-battery/internal/tonproof"
	"github.com/printdreams/cryptocash-ton-battery/internal/user"
)

type TonProofHandler struct {
	Nonces   *nonce.Store
	Verifier tonproof.Config
	JWT      *auth.Issuer
	Users    *user.Store
}

func NewTonProofHandler(n *nonce.Store, verifier tonproof.Config, jwt *auth.Issuer, users *user.Store) *TonProofHandler {
	return &TonProofHandler{Nonces: n, Verifier: verifier, JWT: jwt, Users: users}
}

// Payload godoc
// @Summary      Generate ton_proof payload
// @Description  Returns a single-use payload (nonce) that the wallet must sign for ton_proof login
// @Tags         auth
// @Produce      json
// @Success      200  {object}  handler.PayloadResponse
// @Failure      500  {object}  handler.ErrorResponse
// @Router       /ton-proof/payload [get]
func (h *TonProofHandler) Payload(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if h.Nonces == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "nonce-store-unavailable"})
		return
	}

	payload, err := h.Nonces.Issue(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "nonce-issue-failed"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"payload": payload})
}

// Check godoc
// @Summary      Verify ton_proof and issue a JWT
// @Description  Verifies a ton_proof signature for a v5R1 wallet, resolves the user by public key, and returns a session JWT
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      tonproof.Request  true  "ton_proof payload"
// @Success      200   {object}  handler.TonProofCheckResponse
// @Failure      400   {object}  handler.ErrorResponse
// @Failure      401   {object}  handler.ErrorResponse
// @Failure      500   {object}  handler.ErrorResponse
// @Router       /ton-proof/check [post]
func (h *TonProofHandler) Check(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req tonproof.Request
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16*1024))
	if err := dec.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "malformed-request"})
		return
	}

	if h.Nonces == nil || h.JWT == nil || h.Users == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "auth-not-configured"})
		return
	}

	if err := h.Nonces.Consume(r.Context(), req.Proof.Payload); err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid-nonce"})
		return
	}

	res, err := tonproof.Verify(h.Verifier, req)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
		return
	}

	usr, err := h.Users.FindOrCreate(r.Context(), res.PublicKeyHex)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "user-store-failed"})
		return
	}

	token, exp, err := h.JWT.Issue(usr.UserID)
	if err != nil {
		if errors.Is(err, auth.ErrNotConfigured) {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "jwt-not-configured"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "jwt-issue-failed"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"token":     token,
		"expiresAt": exp,
		"userId":    usr.UserID,
		"publicKey": res.PublicKeyHex,
		"address":   res.AddressRaw,
		"newUser":   usr.Created,
	})
}

func writeJSON(w http.ResponseWriter, status int, body interface{}) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
