package audit

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
)

type Store struct {
	fs *firestore.Client
}

func NewStore(fs *firestore.Client) *Store {
	return &Store{fs: fs}
}

func (s *Store) Append(ctx context.Context, action, actor string, details map[string]interface{}) {
	if s == nil || s.fs == nil {
		return
	}
	_, _, _ = s.fs.Collection("audit_log").Add(ctx, map[string]interface{}{
		"action":     action,
		"actor":      actor,
		"details":    details,
		"created_at": time.Now().UTC(),
	})
}
