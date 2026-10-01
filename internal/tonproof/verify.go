package tonproof

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/ton/wallet"
)

var (
	ErrBadPublicKey     = errors.New("bad-public-key")
	ErrBadAddress       = errors.New("bad-address")
	ErrWrongNetwork     = errors.New("wrong-network")
	ErrNotV5            = errors.New("not-a-v5-wallet")
	ErrDomainNotAllowed = errors.New("domain-not-allowed")
	ErrStaleProof       = errors.New("stale-proof")
	ErrBadSignature     = errors.New("bad-signature")
)

const (
	tonProofPrefix = "ton-proof-item-v2/"
	tonConnectTag  = "ton-connect"
)

type Config struct {
	AllowedDomains    []string
	ProofValidSeconds int
	NetworkGlobalID   int32
}

type Domain struct {
	LengthBytes uint32 `json:"length_bytes"`
	Value       string `json:"value"`
}

type Proof struct {
	Timestamp int64  `json:"timestamp"`
	Domain    Domain `json:"domain"`
	Signature string `json:"signature"`
	Payload   string `json:"payload"`
}

type Request struct {
	Address         string `json:"address"`
	Network         string `json:"network"`
	PublicKey       string `json:"public_key"`
	Proof           Proof  `json:"proof"`
	WalletStateInit string `json:"wallet_state_init"`
}

type Result struct {
	PublicKeyHex string
	AddressRaw   string
}

func Verify(cfg Config, req Request) (*Result, error) {
	pub, err := hex.DecodeString(strings.TrimPrefix(req.PublicKey, "0x"))
	if err != nil || len(pub) != ed25519.PublicKeySize {
		return nil, ErrBadPublicKey
	}

	if req.Network != "" {
		if n, err := strconv.Atoi(strings.TrimSpace(req.Network)); err == nil && int32(n) != cfg.NetworkGlobalID {
			return nil, ErrWrongNetwork
		}
	}

	claimed, err := parseAddress(req.Address)
	if err != nil {
		return nil, ErrBadAddress
	}

	derived, err := DeriveV5Address(ed25519.PublicKey(pub), cfg.NetworkGlobalID, int8(claimed.Workchain()))
	if err != nil {
		return nil, ErrNotV5
	}
	if derived.Workchain() != claimed.Workchain() || !bytes.Equal(derived.Data(), claimed.Data()) {
		return nil, ErrNotV5
	}

	if len(cfg.AllowedDomains) > 0 && !containsDomain(cfg.AllowedDomains, req.Proof.Domain.Value) {
		return nil, ErrDomainNotAllowed
	}

	now := time.Now().Unix()
	if req.Proof.Timestamp > now+60 || now-req.Proof.Timestamp > int64(cfg.ProofValidSeconds) {
		return nil, ErrStaleProof
	}

	sig, err := base64.StdEncoding.DecodeString(req.Proof.Signature)
	if err != nil || len(sig) != ed25519.SignatureSize {
		return nil, ErrBadSignature
	}

	message := BuildMessage(claimed.Workchain(), claimed.Data(), req.Proof.Domain.Value, uint64(req.Proof.Timestamp), []byte(req.Proof.Payload))
	if !verifySignature(pub, message, sig) {
		return nil, ErrBadSignature
	}

	return &Result{
		PublicKeyHex: hex.EncodeToString(pub),
		AddressRaw:   claimed.String(),
	}, nil
}

func BuildMessage(workchain int32, addressHash []byte, domain string, timestamp uint64, payload []byte) []byte {
	var b bytes.Buffer
	b.WriteString(tonProofPrefix)

	wc := make([]byte, 4)
	binary.BigEndian.PutUint32(wc, uint32(workchain))
	b.Write(wc)

	b.Write(addressHash)

	domainBytes := []byte(domain)
	dl := make([]byte, 4)
	binary.LittleEndian.PutUint32(dl, uint32(len(domainBytes)))
	b.Write(dl)
	b.Write(domainBytes)

	ts := make([]byte, 8)
	binary.LittleEndian.PutUint64(ts, timestamp)
	b.Write(ts)

	b.Write(payload)
	return b.Bytes()
}

func Digest(message []byte) []byte {
	inner := sha256.Sum256(message)

	var full bytes.Buffer
	full.Write([]byte{0xff, 0xff})
	full.WriteString(tonConnectTag)
	full.Write(inner[:])

	d := sha256.Sum256(full.Bytes())
	return d[:]
}

func DeriveV5Address(pub ed25519.PublicKey, networkGlobalID int32, workchain int8) (*address.Address, error) {
	return wallet.AddressFromPubKey(pub, wallet.ConfigV5R1Final{
		NetworkGlobalID: networkGlobalID,
		Workchain:       workchain,
	}, 0)
}

func verifySignature(pub ed25519.PublicKey, message, sig []byte) bool {
	return ed25519.Verify(pub, Digest(message), sig)
}

func parseAddress(s string) (*address.Address, error) {
	if strings.Contains(s, ":") {
		return address.ParseRawAddr(s)
	}
	return address.ParseAddr(s)
}

func containsDomain(list []string, v string) bool {
	for _, d := range list {
		if strings.EqualFold(d, v) {
			return true
		}
	}
	return false
}
