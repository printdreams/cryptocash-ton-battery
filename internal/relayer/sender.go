package relayer

import (
	"context"
	"encoding/hex"
	"strings"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-go/tlb"
	lton "github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/wallet"
)

type Sender struct {
	w *wallet.Wallet
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

	return &Sender{w: w}, nil
}

func (s *Sender) Address() string {
	return s.w.WalletAddress().String()
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
