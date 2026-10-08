package abuse

import (
	"context"
	"sync"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	controlCollection = "control"
	killDoc           = "killswitch"
)

type Killswitch struct {
	fs  *firestore.Client
	def bool
	ttl time.Duration

	mu      sync.Mutex
	cached  bool
	loaded  bool
	fetched time.Time
}

func NewKillswitch(fs *firestore.Client, def bool, ttl time.Duration) *Killswitch {
	return &Killswitch{fs: fs, def: def, ttl: ttl}
}

func (k *Killswitch) Paused(ctx context.Context) bool {
	k.mu.Lock()
	if k.loaded && time.Since(k.fetched) < k.ttl {
		v := k.cached
		k.mu.Unlock()
		return v
	}
	k.mu.Unlock()

	v := k.def
	if k.fs != nil {
		snap, err := k.fs.Collection(controlCollection).Doc(killDoc).Get(ctx)
		if err == nil {
			if p, derr := snap.DataAt("paused"); derr == nil {
				if b, ok := p.(bool); ok {
					v = b
				}
			}
		} else if status.Code(err) != codes.NotFound {
			return k.def
		}
	}

	k.mu.Lock()
	k.cached = v
	k.loaded = true
	k.fetched = time.Now()
	k.mu.Unlock()
	return v
}

func (k *Killswitch) SetPaused(ctx context.Context, v bool) error {
	if k.fs != nil {
		if _, err := k.fs.Collection(controlCollection).Doc(killDoc).Set(ctx, map[string]interface{}{
			"paused":     v,
			"updated_at": time.Now().UTC(),
		}, firestore.MergeAll); err != nil {
			return err
		}
	}
	k.mu.Lock()
	k.cached = v
	k.loaded = true
	k.fetched = time.Now()
	k.mu.Unlock()
	return nil
}
