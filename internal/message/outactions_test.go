package message

import (
	"strings"
	"testing"

	"github.com/xssnick/tonutils-go/address"
)

func rawAddr(t *testing.T, hex64 string) *address.Address {
	t.Helper()
	a, err := address.ParseRawAddr("0:" + hex64)
	if err != nil {
		t.Fatalf("parse addr: %v", err)
	}
	return a
}

func TestForwardEligible(t *testing.T) {
	relayer := rawAddr(t, strings.Repeat("a", 64))
	other := rawAddr(t, strings.Repeat("b", 64))
	capNano := int64(100000000) // 0.1 TON

	msgs := []OutMsg{
		{Op: OpJettonTransfer, ValueNano: 50000000, ResponseDest: relayer},  // eligible
		{Op: OpNFTTransfer, ValueNano: 40000000, ResponseDest: relayer},     // eligible
		{Op: OpJettonTransfer, ValueNano: 50000000, ResponseDest: other},    // wrong response dest
		{Op: 0, ValueNano: 50000000, ResponseDest: relayer},                 // plain Gram: never funded
		{Op: OpJettonTransfer, ValueNano: 200000000, ResponseDest: relayer}, // over cap
		{Op: OpJettonTransfer, ValueNano: 50000000, ResponseDest: nil},      // no response dest
	}

	elig := ForwardEligible(msgs, relayer, capNano)
	if len(elig) != 2 {
		t.Fatalf("expected 2 eligible actions, got %d", len(elig))
	}

	var total int64
	for _, m := range elig {
		total += m.ValueNano
	}
	if total != 90000000 {
		t.Fatalf("expected total funded 90000000, got %d", total)
	}
}

func TestForwardEligibleNilRelayer(t *testing.T) {
	msgs := []OutMsg{{Op: OpJettonTransfer, ValueNano: 50000000, ResponseDest: rawAddr(t, strings.Repeat("c", 64))}}
	if got := ForwardEligible(msgs, nil, 100000000); got != nil {
		t.Fatalf("expected nil when relayer is nil, got %v", got)
	}
}
