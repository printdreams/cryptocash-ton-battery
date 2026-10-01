package user

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	usersCollection = "users"
	keysCollection  = "user_keys"
)

type User struct {
	ID        string    `firestore:"-"`
	PublicKey string    `firestore:"public_key"`
	CreatedAt time.Time `firestore:"created_at"`
	UpdatedAt time.Time `firestore:"updated_at"`
}

type Result struct {
	UserID  string
	Created bool
}

type Store struct {
	fs *firestore.Client
}

func NewStore(fs *firestore.Client) *Store {
	return &Store{fs: fs}
}

func (s *Store) FindOrCreate(ctx context.Context, publicKey string) (*Result, error) {
	res := &Result{}
	err := s.fs.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		keyRef := s.fs.Collection(keysCollection).Doc(publicKey)
		snap, err := tx.Get(keyRef)
		if err != nil && status.Code(err) != codes.NotFound {
			return err
		}
		if err == nil && snap.Exists() {
			uid, e := snap.DataAt("user_id")
			if e == nil {
				if existing, ok := uid.(string); ok && existing != "" {
					res.UserID = existing
					res.Created = false
					return nil
				}
			}
		}

		userID := uuid.NewString()
		now := time.Now().UTC()
		userRef := s.fs.Collection(usersCollection).Doc(userID)
		if err := tx.Create(userRef, map[string]interface{}{
			"public_key": publicKey,
			"created_at": now,
			"updated_at": now,
		}); err != nil {
			return err
		}
		if err := tx.Create(keyRef, map[string]interface{}{
			"user_id":    userID,
			"created_at": now,
		}); err != nil {
			return err
		}
		res.UserID = userID
		res.Created = true
		return nil
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (s *Store) Get(ctx context.Context, userID string) (*User, error) {
	snap, err := s.fs.Collection(usersCollection).Doc(userID).Get(ctx)
	if err != nil {
		return nil, err
	}
	var u User
	if err := snap.DataTo(&u); err != nil {
		return nil, err
	}
	u.ID = snap.Ref.ID
	return &u, nil
}
