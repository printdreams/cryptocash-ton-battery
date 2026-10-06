package relayer

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"strings"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-go/tlb"
	lton "github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/wallet"
	"github.com/xssnick/tonutils-go/tvm/cell"

	"github.com/printdreams/cryptocash-ton-battery/internal/tonproof"
)

type Sender struct {
	w   *wallet.Wallet
	net int32
}

func NewSender(ctx context.Context, mnemonic, configURL string, networkGlobalID int32) (*Sender, error) {
	words := strings.Fields(mnemonic)
	if len(words) == 0 {
		return nil, ErrNoMnemonic
	}

	pool := liteclient.NewConnectionPool()
	cfg, err := liteclient.GetConfigFromUrl(ctx, configURL)
	if err != nil {
		return nil, err
	}
	if err := pool.AddConnectionsFromConfig(ctx, cfg); err != nil {
		return nil, err
	}

	api := lton.NewAPIClient(pool, lton.ProofCheckPolicyFast)
	api.SetTrustedBlockFromConfig(cfg)

	w, err := wallet.FromSeedWithOptions(api, words, wallet.ConfigV5R1Final{
		NetworkGlobalID: networkGlobalID,
	})
	if err != nil {
		return nil, err
	}

	return &Sender{w: w, net: networkGlobalID}, nil
}

func (s *Sender) Address() string {
	return s.w.WalletAddress().String()
}

func (s *Sender) RelayUserMessage(ctx context.Context, userPubKey, bodyBOC []byte, gasTON string) (string, error) {
	pub := ed25519.PublicKey(userPubKey)

	userAddr, err := tonproof.DeriveV5Address(pub, s.net, 0)
	if err != nil {
		return "", err
	}

	stateInit, err := wallet.GetStateInit(pub, wallet.ConfigV5R1Final{NetworkGlobalID: s.net}, 0)
	if err != nil {
		return "", err
	}

	body, err := cell.FromBOC(bodyBOC)
	if err != nil {
		return "", err
	}

	amount, err := tlb.FromTON(gasTON)
	if err != nil {
		return "", err
	}

	msg := &wallet.Message{
		Mode: 3,
		InternalMessage: &tlb.InternalMessage{
			IHRDisabled: true,
			Bounce:      false,
			DstAddr:     userAddr,
			Amount:      amount,
			StateInit:   stateInit,
			Body:        body,
		},
	}

	tx, _, err := s.w.SendWaitTransaction(ctx, msg)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(tx.Hash), nil
}

func (s *Sender) Send(ctx context.Context, to, amountTON, comment string) (string, error) {
	addr, err := address.ParseAddr(to)
	if err != nil {
		addr, err = address.ParseRawAddr(to)
		if err != nil {
			return "", err
		}
	}

	amount, err := tlb.FromTON(amountTON)
	if err != nil {
		return "", err
	}

	transfer, err := s.w.BuildTransfer(addr, amount, false, comment)
	if err != nil {
		return "", err
	}

	tx, _, err := s.w.SendWaitTransaction(ctx, transfer)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(tx.Hash), nil
}
