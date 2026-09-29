package firebase

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"google.golang.org/api/option"

	"github.com/printdreams/cryptocash-ton-battery/internal/config"
)

type Clients struct {
	App       *firebase.App
	Auth      *auth.Client
	Firestore *firestore.Client
}

func Init(ctx context.Context, cfg *config.Config) (*Clients, error) {
	credsJSON, err := cfg.ServiceAccountJSON()
	if err != nil {
		return nil, fmt.Errorf("build service account credentials: %w", err)
	}

	opt := option.WithCredentialsJSON(credsJSON)

	app, err := firebase.NewApp(ctx, &firebase.Config{ProjectID: cfg.ProjectID}, opt)
	if err != nil {
		return nil, fmt.Errorf("initialize firebase app: %w", err)
	}

	authClient, err := app.Auth(ctx)
	if err != nil {
		return nil, fmt.Errorf("initialize firebase auth client: %w", err)
	}

	firestoreClient, err := app.Firestore(ctx)
	if err != nil {
		return nil, fmt.Errorf("initialize firestore client: %w", err)
	}

	return &Clients{App: app, Auth: authClient, Firestore: firestoreClient}, nil
}
