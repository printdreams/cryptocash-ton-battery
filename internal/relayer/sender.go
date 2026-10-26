package relayer

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
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

func TONStringToNano(s string) int64 {
	c, err := tlb.FromTON(s)
	if err != nil {
		return 0
	}
	return c.Nano().Int64()
}

func (s *Sender) buildRelayMessage(userPubKey, bodyBOC []byte, gasNano int64) (*wallet.Message, error) {
	pub := ed25519.PublicKey(userPubKey)

	userAddr, err := tonproof.DeriveV5Address(pub, s.net, 0)
	if err != nil {
		return nil, err
	}

	stateInit, err := wallet.GetStateInit(pub, wallet.ConfigV5R1Final{NetworkGlobalID: s.net}, 0)
	if err != nil {
		return nil, err
	}

	body, err := cell.FromBOC(bodyBOC)
	if err != nil {
		return nil, err
	}

	if gasNano < 0 {
		gasNano = 0
	}

	return &wallet.Message{
		Mode: 3,
		InternalMessage: &tlb.InternalMessage{
			IHRDisabled: true,
			Bounce:      false,
			DstAddr:     userAddr,
			Amount:      tlb.FromNanoTONU(uint64(gasNano)),
			StateInit:   stateInit,
			Body:        body,
		},
	}, nil
}

func (s *Sender) RelayUserMessage(ctx context.Context, userPubKey, bodyBOC []byte, gasNano int64) (string, error) {
	msg, err := s.buildRelayMessage(userPubKey, bodyBOC, gasNano)
	if err != nil {
		return "", err
	}

	tx, _, err := s.w.SendWaitTransaction(ctx, msg)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(tx.Hash), nil
}

func (s *Sender) BuildRelayExternalBOC(ctx context.Context, userPubKey, bodyBOC []byte, gasNano int64) (string, error) {
	msg, err := s.buildRelayMessage(userPubKey, bodyBOC, gasNano)
	if err != nil {
		return "", err
	}

	ext, err := s.w.BuildExternalMessage(ctx, msg)
	if err != nil {
		return "", err
	}

	c, err := ext.ToCell()
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(c.ToBOC()), nil
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
