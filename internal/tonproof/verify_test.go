package tonproof

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"testing"
	"time"

	"github.com/xssnick/tonutils-go/ton/wallet"
)

func signProof(priv ed25519.PrivateKey, workchain int32, addressHash []byte, domain, payload string, ts int64) string {
	msg := BuildMessage(workchain, addressHash, domain, uint64(ts), []byte(payload))
	inner := sha256.Sum256(msg)
	full := []byte{0xff, 0xff}
	full = append(full, []byte(tonConnectTag)...)
	full = append(full, inner[:]...)
	digest := sha256.Sum256(full)
	return base64.StdEncoding.EncodeToString(ed25519.Sign(priv, digest[:]))
}

func TestVerifyRoundTrip(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	netID := int32(-3)

	addr, err := wallet.AddressFromPubKey(pub, wallet.ConfigV5R1Final{NetworkGlobalID: netID, Workchain: 0}, 0)
	if err != nil {
		t.Fatalf("derive v5 address: %v", err)
	}

	ts := time.Now().Unix()
	domain := "localhost"
	payload := "deadbeefdeadbeef"
	sig := signProof(priv, addr.Workchain(), addr.Data(), domain, payload, ts)

	req := Request{
		Address:   addr.String(),
		Network:   "-3",
		PublicKey: hex.EncodeToString(pub),
		Proof: Proof{
			Timestamp: ts,
			Domain:    Domain{LengthBytes: uint32(len(domain)), Value: domain},
			Signature: sig,
			Payload:   payload,
		},
	}
	cfg := Config{ProofValidSeconds: 900, NetworkGlobalID: netID}

	if _, err := Verify(cfg, req); err != nil {
		t.Fatalf("expected valid proof, got %v", err)
	}

	tampered := req
	tampered.Proof.Payload = "0000000000000000"
	if _, err := Verify(cfg, tampered); err == nil {
		t.Fatal("expected failure on tampered payload")
	}

	stale := req
	stale.Proof.Timestamp = ts - 100000
	stale.Proof.Signature = signProof(priv, addr.Workchain(), addr.Data(), domain, payload, stale.Proof.Timestamp)
	if _, err := Verify(cfg, stale); err != ErrStaleProof {
		t.Fatalf("expected stale-proof, got %v", err)
	}
}
