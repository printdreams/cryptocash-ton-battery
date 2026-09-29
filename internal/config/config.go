package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port                    string
	ServiceAccountType      string
	ProjectID               string
	PrivateKeyID            string
	PrivateKey              string
	ClientEmail             string
	ClientID                string
	AuthURI                 string
	TokenURI                string
	AuthProviderX509CertURL string
	ClientX509CertURL       string
	UniverseDomain          string
}

type serviceAccount struct {
	Type                    string `json:"type"`
	ProjectID               string `json:"project_id"`
	PrivateKeyID            string `json:"private_key_id"`
	PrivateKey              string `json:"private_key"`
	ClientEmail             string `json:"client_email"`
	ClientID                string `json:"client_id"`
	AuthURI                 string `json:"auth_uri"`
	TokenURI                string `json:"token_uri"`
	AuthProviderX509CertURL string `json:"auth_provider_x509_cert_url"`
	ClientX509CertURL       string `json:"client_x509_cert_url"`
	UniverseDomain          string `json:"universe_domain"`
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		Port:                    getEnvDefault("PORT", "8080"),
		ServiceAccountType:      getEnvDefault("SERVICE_ACCOUNT", "service_account"),
		ProjectID:               os.Getenv("PROJECT_ID"),
		PrivateKeyID:            os.Getenv("PRIVATE_KEY_ID"),
		PrivateKey:              normalizeKey(os.Getenv("PRIVATE_KEY_FIREBASE")),
		ClientEmail:             os.Getenv("CLIENT_EMAIL"),
		ClientID:                os.Getenv("CLIENT_ID"),
		AuthURI:                 getEnvDefault("AUTH_URI", "https://accounts.google.com/o/oauth2/auth"),
		TokenURI:                getEnvDefault("TOKEN_URI", "https://oauth2.googleapis.com/token"),
		AuthProviderX509CertURL: getEnvDefault("AUTH_PROVIDER_X509_CERT_URL", "https://www.googleapis.com/oauth2/v1/certs"),
		ClientX509CertURL:       os.Getenv("CLIENT_X509_CERT_URL"),
		UniverseDomain:          getEnvDefault("UNIVERSE_DOMAIN", "googleapis.com"),
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) validate() error {
	missing := make([]string, 0)
	required := map[string]string{
		"PROJECT_ID":           c.ProjectID,
		"PRIVATE_KEY_ID":       c.PrivateKeyID,
		"PRIVATE_KEY_FIREBASE": c.PrivateKey,
		"CLIENT_EMAIL":         c.ClientEmail,
		"CLIENT_ID":            c.ClientID,
	}
	for name, value := range required {
		if strings.TrimSpace(value) == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}
	return nil
}

func (c *Config) ServiceAccountJSON() ([]byte, error) {
	sa := serviceAccount{
		Type:                    c.ServiceAccountType,
		ProjectID:               c.ProjectID,
		PrivateKeyID:            c.PrivateKeyID,
		PrivateKey:              c.PrivateKey,
		ClientEmail:             c.ClientEmail,
		ClientID:                c.ClientID,
		AuthURI:                 c.AuthURI,
		TokenURI:                c.TokenURI,
		AuthProviderX509CertURL: c.AuthProviderX509CertURL,
		ClientX509CertURL:       c.ClientX509CertURL,
		UniverseDomain:          c.UniverseDomain,
	}
	return json.Marshal(sa)
}

func normalizeKey(key string) string {
	return strings.ReplaceAll(key, `\n`, "\n")
}

func getEnvDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
