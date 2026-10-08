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
	MessageMaxBytes         int
	MessageMinTTLSeconds    int
	TonEmulateURL           string
	PolicyMaxOutMessages    int
	PolicyFeeCapNano        int64
	PolicyNanoPerCharge     int64
	PolicyMarginCharges     int64
	PolicyMinCharge         int64
	PolicyBlockedDest       []string
	RelayGasTON             string
	PriceBaseURL            string
	PriceCoinID             string
	PriceCurrency           string
	PriceCentsPerCharge     int64
	PriceTTLSeconds         int
	PrintBufferBaseNano     int64
	PrintMarginBPS          int64
	PrintQuoteTTLSeconds    int
	RateLimitPerMin         int
	VelocityPerDay          int
	BatteryPaused           bool
	KillswitchTTLSeconds    int
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
		MessageMaxBytes:         getEnvInt("MESSAGE_MAX_BYTES", 8192),
		MessageMinTTLSeconds:    getEnvInt("MESSAGE_MIN_TTL_SECONDS", 60),
		TonEmulateURL:           getEnvDefault("TON_EMULATE_URL", "https://testnet.tonapi.io"),
		PolicyMaxOutMessages:    getEnvInt("POLICY_MAX_OUT_MESSAGES", 4),
		PolicyFeeCapNano:        getEnvInt64("POLICY_FEE_CAP_NANO", 1000000000),
		PolicyNanoPerCharge:     getEnvInt64("POLICY_NANO_PER_CHARGE", 10000000),
		PolicyMarginCharges:     getEnvInt64("POLICY_MARGIN_CHARGES", 1),
		PolicyMinCharge:         getEnvInt64("POLICY_MIN_CHARGE", 1),
		PolicyBlockedDest:       getEnvList("POLICY_BLOCKED_DESTINATIONS"),
		RelayGasTON:             getEnvDefault("RELAY_GAS_TON", "0.1"),
		PriceBaseURL:            getEnvDefault("PRICE_BASE_URL", "https://api.coingecko.com"),
		PriceCoinID:             getEnvDefault("PRICE_COIN_ID", "the-open-network"),
		PriceCurrency:           getEnvDefault("PRICE_CURRENCY", "usd"),
		PriceCentsPerCharge:     getEnvInt64("PRICE_CENTS_PER_CHARGE", 1),
		PriceTTLSeconds:         getEnvInt("PRICE_TTL_SECONDS", 300),
		PrintBufferBaseNano:     getEnvInt64("PRINT_BUFFER_BASE_NANO", 50000000),
		PrintMarginBPS:          getEnvInt64("PRINT_MARGIN_BPS", 15000),
		PrintQuoteTTLSeconds:    getEnvInt("PRINT_QUOTE_TTL_SECONDS", 300),
		RateLimitPerMin:         getEnvInt("RATE_LIMIT_PER_MIN", 30),
		VelocityPerDay:          getEnvInt("VELOCITY_PER_DAY", 500),
		BatteryPaused:           getEnvBool("BATTERY_PAUSED", false),
		KillswitchTTLSeconds:    getEnvInt("KILLSWITCH_TTL_SECONDS", 10),
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

func getEnvInt64(key string, fallback int64) int64 {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64); err == nil {
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
