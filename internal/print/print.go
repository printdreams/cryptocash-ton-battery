package print

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const collection = "print_jobs"

var ErrNotFound = errors.New("print-job-not-found")

type Config struct {
	BufferBaseNano int64
	MarginBPS      int64
	QuoteTTL       time.Duration
}

func BufferNano(cfg Config) int64 {
	if cfg.MarginBPS <= 0 {
		return cfg.BufferBaseNano
	}
	return cfg.BufferBaseNano * cfg.MarginBPS / 10000
}

func NanoToTON(n int64) string {
	whole := n / 1000000000
	frac := n % 1000000000
	s := fmt.Sprintf("%d.%09d", whole, frac)
	i := len(s) - 1
	for i >= 0 && s[i] == '0' {
		i--
	}
	if i >= 0 && s[i] == '.' {
		i--
	}
	return s[:i+1]
}

type Job struct {
	ID         string    `firestore:"-"`
	UserID     string    `firestore:"user_id"`
	ToAddress  string    `firestore:"to_address"`
	BufferNano int64     `firestore:"buffer_nano"`
	Charge     int64     `firestore:"charge"`
	Status     string    `firestore:"status"`
	TxHash     string    `firestore:"tx_hash"`
	CreatedAt  time.Time `firestore:"created_at"`
	ExpiresAt  time.Time `firestore:"expires_at"`
}

type Store struct {
	fs *firestore.Client
}

func NewStore(fs *firestore.Client) *Store {
	return &Store{fs: fs}
}

func (s *Store) Create(ctx context.Context, job *Job) (string, error) {
	id := uuid.NewString()
	job.ID = id
	if _, err := s.fs.Collection(collection).Doc(id).Set(ctx, job); err != nil {
		return "", err
	}
	return id, nil
}

func (s *Store) Get(ctx context.Context, id string) (*Job, error) {
	snap, err := s.fs.Collection(collection).Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	var j Job
	if err := snap.DataTo(&j); err != nil {
		return nil, err
	}
	j.ID = snap.Ref.ID
	return &j, nil
}

func (s *Store) Update(ctx context.Context, id string, fields map[string]interface{}) error {
	_, err := s.fs.Collection(collection).Doc(id).Set(ctx, fields, firestore.MergeAll)
	return err
}
