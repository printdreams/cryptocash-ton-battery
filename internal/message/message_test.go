package message

import (
	"encoding/base64"
	"testing"
	"time"

	"github.com/xssnick/tonutils-go/tvm/cell"
)

func buildEnvelope(op, walletID, validUntil, seqno uint64) string {
	c := cell.BeginCell().
		MustStoreUInt(op, 32).
		MustStoreUInt(walletID, 32).
		MustStoreUInt(validUntil, 32).
		MustStoreUInt(seqno, 32).
		EndCell()
	return base64.StdEncoding.EncodeToString(c.ToBOC())
}

func TestCheckValid(t *testing.T) {
	now := time.Now().Unix()
	boc := buildEnvelope(authSignedInternal, 2147483645, uint64(now+300), 7)
	cfg := Config{MaxBytes: 8192, MinTTLSeconds: 60}

	p, err := Check(boc, cfg, now)
	if err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
	if p.OpCode != authSignedInternal || p.Seqno != 7 {
		t.Fatalf("parsed wrong: %+v", p)
	}
}

func TestCheckUnsupportedOp(t *testing.T) {
	now := time.Now().Unix()
	boc := buildEnvelope(0x7369676e, 1, uint64(now+300), 0)
	if _, err := Check(boc, Config{MaxBytes: 8192, MinTTLSeconds: 60}, now); err != ErrUnsupportedOp {
		t.Fatalf("expected unsupported-operation, got %v", err)
	}
}

func TestCheckTTLTooShort(t *testing.T) {
	now := time.Now().Unix()
	boc := buildEnvelope(authSignedInternal, 1, uint64(now+10), 0)
	if _, err := Check(boc, Config{MaxBytes: 8192, MinTTLSeconds: 60}, now); err != ErrTTLTooShort {
		t.Fatalf("expected ttl-too-short, got %v", err)
	}
}

func TestCheckTooLarge(t *testing.T) {
	now := time.Now().Unix()
	boc := buildEnvelope(authSignedInternal, 1, uint64(now+300), 0)
	if _, err := Check(boc, Config{MaxBytes: 4, MinTTLSeconds: 60}, now); err != ErrTooLarge {
		t.Fatalf("expected message-too-large, got %v", err)
	}
}

func TestCheckMalformed(t *testing.T) {
	now := time.Now().Unix()
	if _, err := Check("!!!not-base64!!!", Config{MaxBytes: 8192, MinTTLSeconds: 60}, now); err != ErrMalformed {
		t.Fatalf("expected malformed-message, got %v", err)
	}
}
