package handler

import (
	"context"

	"github.com/printdreams/cryptocash-ton-battery/internal/emulate"
	"github.com/printdreams/cryptocash-ton-battery/internal/message"
	"github.com/printdreams/cryptocash-ton-battery/internal/relayer"
)

func computeFundedForward(ctx context.Context, em *emulate.Client, sender *relayer.Sender, pub []byte, boc string, capNano int64, checkOwner bool) int64 {
	if sender == nil {
		return 0
	}
	msgs, err := message.ParseOutMsgs(boc)
	if err != nil || len(msgs) == 0 {
		return 0
	}
	relayerAddr := sender.WalletAddress()
	elig := message.ForwardEligible(msgs, relayerAddr, capNano)
	if len(elig) == 0 {
		return 0
	}

	var total int64
	for _, m := range elig {
		if checkOwner {
			if em == nil {
				return 0
			}
			userAddr, uerr := sender.UserAddress(pub)
			if uerr != nil {
				return 0
			}
			owned := false
			switch m.Op {
			case message.OpJettonTransfer:
				owned, _ = em.OwnsJettonWallet(ctx, userAddr.StringRaw(), m.Dest.StringRaw())
			case message.OpNFTTransfer:
				owned, _ = em.OwnsNFT(ctx, userAddr.StringRaw(), m.Dest.StringRaw())
			}
			if !owned {
				continue
			}
		}
		total += m.ValueNano
	}
	return total
}
