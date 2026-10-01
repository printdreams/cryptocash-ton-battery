package handler

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"strconv"
	"time"

	"github.com/printdreams/cryptocash-ton-battery/internal/tonproof"
)

type DevSignHandler struct {
	Verifier tonproof.Config
}

func NewDevSignHandler(verifier tonproof.Config) *DevSignHandler {
	return &DevSignHandler{Verifier: verifier}
}

// Sign godoc
// @Summary      [TO BE DELETED] Dev-only ton_proof signer
// @Description  TEST ONLY — signs a ton_proof for a fixed test wallet from a payload, so /ton-proof/check can be exercised without a real wallet. Enabled only when DEV_SIGN=true. Response carries header X-Dev-Route: TO-BE-DELETED.
// @Tags         dev
// @Produce      json
// @Param        payload  query     string  true   "payload from /ton-proof/payload"
// @Param        domain   query     string  false  "app domain (default localhost)"
// @Success      200      {object}  tonproof.Request
// @Failure      400      {object}  map[string]string
// @Router       /ton-proof/dev-sign [get]
func (h *DevSignHandler) Sign(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Dev-Route", "TO-BE-DELETED")

	payload := r.URL.Query().Get("payload")
	if payload == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing-payload"})
		return
	}
	domain := r.URL.Query().Get("domain")
	if domain == "" {
		domain = "localhost"
	}

	seed := sha256.Sum256([]byte("cryptocash-ton-battery-dev-wallet"))
	priv := ed25519.NewKeyFromSeed(seed[:])
	pub := priv.Public().(ed25519.PublicKey)

	addr, err := tonproof.DeriveV5Address(pub, h.Verifier.NetworkGlobalID, 0)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "derive-failed"})
		return
	}

	ts := time.Now().Unix()
	msg := tonproof.BuildMessage(addr.Workchain(), addr.Data(), domain, uint64(ts), []byte(payload))
	sig := ed25519.Sign(priv, tonproof.Digest(msg))

	body := tonproof.Request{
		Address:   addr.String(),
		Network:   strconv.Itoa(int(h.Verifier.NetworkGlobalID)),
		PublicKey: hex.EncodeToString(pub),
		Proof: tonproof.Proof{
			Timestamp: ts,
			Domain:    tonproof.Domain{LengthBytes: uint32(len(domain)), Value: domain},
			Signature: base64.StdEncoding.EncodeToString(sig),
			Payload:   payload,
		},
	}

	writeJSON(w, http.StatusOK, body)
}
