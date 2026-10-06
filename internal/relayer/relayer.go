package relayer

import (
	"crypto/ed25519"
	"errors"
	"strings"

	"github.com/xssnick/tonutils-go/ton/wallet"

	"github.com/printdreams/cryptocash-ton-battery/internal/tonproof"
)

var ErrNoMnemonic = errors.New("relayer mnemonic not configured")

type Relayer struct {
	priv    ed25519.PrivateKey
	pub     ed25519.PublicKey
	address string
}

func Load(mnemonic string, networkGlobalID int32) (*Relayer, error) {
	words := strings.Fields(mnemonic)
	if len(words) == 0 {
		return nil, ErrNoMnemonic
	}

	priv, err := wallet.SeedToPrivateKeyWithOptions(words)
	if err != nil {
		return nil, err
	}
	pub := priv.Public().(ed25519.PublicKey)

	addr, err := tonproof.DeriveV5Address(pub, networkGlobalID, 0)
	if err != nil {
		return nil, err
	}

	return &Relayer{priv: priv, pub: pub, address: addr.String()}, nil
}

func (r *Relayer) Address() string {
	return r.address
}

func (r *Relayer) PublicKey() ed25519.PublicKey {
	return r.pub
}
