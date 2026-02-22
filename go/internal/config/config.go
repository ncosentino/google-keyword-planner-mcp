// Package config resolves Google Ads API credentials from multiple sources.
// Priority order: CLI flags > environment variables > .env file.
//
// Required credentials:
//   - Developer token: GOOGLE_ADS_DEVELOPER_TOKEN
//   - OAuth2 client ID: GOOGLE_ADS_CLIENT_ID
//   - OAuth2 client secret: GOOGLE_ADS_CLIENT_SECRET
//   - OAuth2 refresh token: GOOGLE_ADS_REFRESH_TOKEN
//   - Google Ads customer ID: GOOGLE_ADS_CUSTOMER_ID
//
// Optional credentials:
//   - Login customer ID: GOOGLE_ADS_LOGIN_CUSTOMER_ID (manager account ID; required when
//     GOOGLE_ADS_CUSTOMER_ID is a sub-account accessed through a manager/MCC account)
package config

import (
	"bufio"
	"log/slog"
	"os"
	"strings"
)

const (
	envDeveloperToken  = "GOOGLE_ADS_DEVELOPER_TOKEN"
	envClientID        = "GOOGLE_ADS_CLIENT_ID"
	envClientSecret    = "GOOGLE_ADS_CLIENT_SECRET"
	envRefreshToken    = "GOOGLE_ADS_REFRESH_TOKEN"
	envCustomerID      = "GOOGLE_ADS_CUSTOMER_ID"
	envLoginCustomerID = "GOOGLE_ADS_LOGIN_CUSTOMER_ID"
	dotEnvFile         = ".env"
)

// Config holds resolved Google Ads API credentials.
type Config struct {
	// DeveloperToken is the Google Ads developer token.
	DeveloperToken string
	// ClientID is the OAuth2 client ID.
	ClientID string
	// ClientSecret is the OAuth2 client secret.
	ClientSecret string
	// RefreshToken is the pre-obtained OAuth2 refresh token.
	RefreshToken string
	// CustomerID is the Google Ads customer account ID (digits only, e.g. "1234567890").
	CustomerID string
	// LoginCustomerID is the manager/MCC account ID used as the login-customer-id header.
	// Required when CustomerID is a sub-account accessed through a manager account.
	LoginCustomerID string
}

// Flags holds values parsed from CLI flags.
type Flags struct {
	DeveloperToken  string
	ClientID        string
	ClientSecret    string
	RefreshToken    string
	CustomerID      string
	LoginCustomerID string
}

// IsComplete returns true when all required fields are populated.
func (c Config) IsComplete() bool {
	return c.DeveloperToken != "" && c.ClientID != "" &&
		c.ClientSecret != "" && c.RefreshToken != "" && c.CustomerID != ""
}

// Resolve returns a Config populated from flags, then environment variables, then .env file.
// Each field is resolved independently from the highest-priority non-empty source.
func Resolve(flags Flags) Config {
	dotenv := parseDotEnv()

	return Config{
		DeveloperToken:  resolve("developer token", flags.DeveloperToken, envDeveloperToken, dotenv),
		ClientID:        resolve("client ID", flags.ClientID, envClientID, dotenv),
		ClientSecret:    resolve("client secret", flags.ClientSecret, envClientSecret, dotenv),
		RefreshToken:    resolve("refresh token", flags.RefreshToken, envRefreshToken, dotenv),
		CustomerID:      normalizeCustomerID(resolve("customer ID", flags.CustomerID, envCustomerID, dotenv)),
		LoginCustomerID: normalizeCustomerID(resolve("login customer ID", flags.LoginCustomerID, envLoginCustomerID, dotenv)),
	}
}

func resolve(name, flagVal, envVar string, dotenv map[string]string) string {
	if flagVal != "" {
		slog.Debug("credential loaded from CLI flag", "field", name)
		return flagVal
	}
	if v := os.Getenv(envVar); v != "" {
		slog.Debug("credential loaded from environment variable", "field", name, "env", envVar)
		return v
	}
	if v, ok := dotenv[envVar]; ok && v != "" {
		slog.Debug("credential loaded from .env file", "field", name, "env", envVar)
		return v
	}
	return ""
}

// normalizeCustomerID strips dashes from a customer ID (123-456-7890 -> 1234567890).
func normalizeCustomerID(id string) string {
	return strings.ReplaceAll(id, "-", "")
}

func parseDotEnv() map[string]string {
	result := make(map[string]string)
	f, err := os.Open(dotEnvFile)
	if err != nil {
		return result
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		result[strings.TrimSpace(k)] = strings.Trim(strings.TrimSpace(v), `"'`)
	}
	return result
}
