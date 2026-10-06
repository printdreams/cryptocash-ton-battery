package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
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
	NonceTTLSeconds         int
	AllowedDomains          []string
	ProofValidSeconds       int
	NetworkGlobalID         int
	JWTSecret               string
	JWTTTLSeconds           int
	AdminToken              string
	DevSignEnabled          bool
	TonEndpoint             string
	TonAPIKey               string
	RelayerMnemonic         string
	TonConfigURL            string
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
		NonceTTLSeconds:         getEnvInt("NONCE_TTL_SECONDS", 600),
		AllowedDomains:          getEnvList("ALLOWED_DOMAINS"),
		ProofValidSeconds:       getEnvInt("PROOF_VALID_SECONDS", 900),
		NetworkGlobalID:         getEnvInt("TON_NETWORK_GLOBAL_ID", -3),
		JWTSecret:               os.Getenv("JWT_SECRET"),
		JWTTTLSeconds:           getEnvInt("JWT_TTL_SECONDS", 86400),
		AdminToken:              os.Getenv("ADMIN_TOKEN"),
		DevSignEnabled:          getEnvBool("DEV_SIGN", false),
		TonEndpoint:             getEnvDefault("TON_ENDPOINT", "https://testnet.toncenter.com"),
		TonAPIKey:               os.Getenv("TON_API_KEY"),
		RelayerMnemonic:         os.Getenv("RELAYER_MNEMONIC"),
		TonConfigURL:            getEnvDefault("TON_CONFIG_URL", "https://ton.org/testnet-global.config.json"),
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

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			return n
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(strings.TrimSpace(v)); err == nil {
			return b
		}
	}
	return fallback
}

func getEnvList(key string) []string {
	raw := os.Getenv(key)
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
