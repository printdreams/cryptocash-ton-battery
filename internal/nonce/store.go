package nonce

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const collection = "ton_proof_nonces"

var ErrInvalidNonce = errors.New("invalid or expired nonce")

type Store struct {
	fs  *firestore.Client
	ttl time.Duration
}

func NewStore(fs *firestore.Client, ttl time.Duration) *Store {
	return &Store{fs: fs, ttl: ttl}
}

func (s *Store) Issue(ctx context.Context) (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	payload := hex.EncodeToString(buf)
	now := time.Now().UTC()
	_, err := s.fs.Collection(collection).Doc(payload).Set(ctx, map[string]interface{}{
		"created_at": now,
		"expires_at": now.Add(s.ttl),
	})
	if err != nil {
		return "", err
	}
	return payload, nil
}

func (s *Store) Consume(ctx context.Context, payload string) error {
	if payload == "" {
		return ErrInvalidNonce
	}
	doc := s.fs.Collection(collection).Doc(payload)
	return s.fs.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		snap, err := tx.Get(doc)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return ErrInvalidNonce
			}
			return err
		}
		exp, err := snap.DataAt("expires_at")
		if err != nil {
			_ = tx.Delete(doc)
			return ErrInvalidNonce
		}
		if t, ok := exp.(time.Time); ok && time.Now().UTC().After(t) {
			_ = tx.Delete(doc)
			return ErrInvalidNonce
		}
		return tx.Delete(doc)
	})
}
