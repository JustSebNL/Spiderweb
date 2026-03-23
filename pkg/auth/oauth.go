package auth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type OAuthProviderConfig struct {
	Issuer       string
	ClientID     string
	ClientSecret string
	TokenURL     string
	Scopes       string
	Originator   string
	Port         int
}

func OpenAIOAuthConfig() OAuthProviderConfig {
	clientID := os.Getenv("OPENAI_OAUTH_CLIENT_ID")
	if clientID == "" {
		clientID = "app_EMoamEEZ73f0CkXaXp7hrann"
	}
	return OAuthProviderConfig{
		Issuer:     "https://auth.openai.com",
		ClientID:   clientID,
		Scopes:     "openid profile email offline_access",
		Originator: "codex_cli_rs",
		Port:       1455,
	}
}

func GoogleAntigravityOAuthConfig() OAuthProviderConfig {
	return OAuthProviderConfig{
		Issuer:       "https://accounts.google.com/o/oauth2/v2",
		TokenURL:     "https://oauth2.googleapis.com/token",
		ClientID:     os.Getenv("GOOGLE_AG_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_AG_CLIENT_SECRET"),
		Scopes:       "https://www.googleapis.com/auth/cloud-platform https://www.googleapis.com/auth/userinfo.email https://www.googleapis.com/auth/userinfo.profile https://www.googleapis.com/auth/cclog https://www.googleapis.com/auth/experimentsandconfigs",
		Port:         51121,
	}
}

func BuildAuthorizeURL(cfg OAuthProviderConfig, pkce PKCECodes, state string, redirectURI string) string {
	u, _ := url.Parse(cfg.Issuer + "/oauth/authorize")
	q := u.Query()
	q.Set("client_id", cfg.ClientID)
	q.Set("response_type", "code")
	q.Set("redirect_uri", redirectURI)
	if cfg.Scopes != "" {
		q.Set("scope", cfg.Scopes)
	}
	q.Set("code_challenge", pkce.CodeChallenge)
	q.Set("code_challenge_method", "S256")
	q.Set("state", state)
	// OpenAI extras validated by tests
	q.Set("id_token_add_organizations", "true")
	q.Set("codex_cli_simplified_flow", "true")
	if cfg.Originator != "" {
		q.Set("originator", cfg.Originator)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func exchangeCodeForTokens(cfg OAuthProviderConfig, code string, verifier string, redirectURI string) (*AuthCredential, error) {
	tokenURL := cfg.TokenURL
	if tokenURL == "" {
		tokenURL = cfg.Issuer + "/oauth/token"
	}
	form := url.Values{
		"grant_type":   {"authorization_code"},
		"client_id":    {cfg.ClientID},
		"code":         {code},
		"redirect_uri": {redirectURI},
		"code_verifier": {verifier},
	}
	if cfg.ClientSecret != "" {
		form.Set("client_secret", cfg.ClientSecret)
	}
	resp, err := http.PostForm(tokenURL, form)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}
	return parseTokenResponseMap(body, "openai")
}

func RefreshAccessToken(cred *AuthCredential, cfg OAuthProviderConfig) (*AuthCredential, error) {
	if cred.RefreshToken == "" {
		return nil, fmt.Errorf("missing refresh token")
	}
	tokenURL := cfg.TokenURL
	if tokenURL == "" {
		tokenURL = cfg.Issuer + "/oauth/token"
	}
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {cfg.ClientID},
		"refresh_token": {cred.RefreshToken},
	}
	if cfg.ClientSecret != "" {
		form.Set("client_secret", cfg.ClientSecret)
	}
	resp, err := http.PostForm(tokenURL, form)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}
	refreshed, err := parseTokenResponseMap(body, cred.Provider)
	if err != nil {
		return nil, err
	}
	if refreshed.RefreshToken == "" {
		refreshed.RefreshToken = cred.RefreshToken
	}
	if refreshed.AccountID == "" {
		refreshed.AccountID = cred.AccountID
	}
	return refreshed, nil
}

func parseTokenResponse(body []byte, provider string) (*AuthCredential, error) {
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, err
	}
	return parseTokenResponseMap(m, provider)
}

func parseTokenResponseMap(m map[string]any, provider string) (*AuthCredential, error) {
	access, _ := m["access_token"].(string)
	if access == "" {
		return nil, fmt.Errorf("missing access_token")
	}
	refresh, _ := m["refresh_token"].(string)
	expiresIn := int64(0)
	switch v := m["expires_in"].(type) {
	case float64:
		expiresIn = int64(v)
	case int64:
		expiresIn = v
	}
	cred := &AuthCredential{
		AccessToken: access,
		RefreshToken: refresh,
		Provider: provider,
		AuthMethod: "oauth",
	}
	if expiresIn > 0 {
		cred.ExpiresAt = time.Now().Add(time.Duration(expiresIn) * time.Second)
	}
	if idToken, _ := m["id_token"].(string); idToken != "" {
		if acc := extractAccountID(idToken); acc != "" {
			cred.AccountID = acc
		}
	}
	return cred, nil
}

func extractAccountID(idToken string) string {
	parts := strings.Split(idToken, ".")
	if len(parts) < 2 {
		return ""
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return ""
	}
	// Preferred path used in tests
	if sub, ok := claims["https://api.openai.com/auth"].(map[string]any); ok {
		if acc, _ := sub["chatgpt_account_id"].(string); acc != "" {
			return acc
		}
	}
	// Direct top-level claim fallback
	if acc, _ := claims["chatgpt_account_id"].(string); acc != "" {
		return acc
	}
	// Fallback organizations
	if orgs, ok := claims["organizations"].([]any); ok {
		for _, o := range orgs {
			if m, ok := o.(map[string]any); ok {
				if id, _ := m["id"].(string); id != "" {
					return id
				}
			}
		}
	}
	return ""
}

type DeviceCodeResponse struct {
	DeviceAuthID string
	UserCode     string
	Interval     int
}

func parseDeviceCodeResponse(body []byte) (*DeviceCodeResponse, error) {
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, err
	}
	interval := 0
	switch v := m["interval"].(type) {
	case float64:
		interval = int(v)
	case string:
		if i, err := strconv.Atoi(v); err == nil {
			interval = i
		} else {
			return nil, fmt.Errorf("invalid interval")
		}
	}
	return &DeviceCodeResponse{
		DeviceAuthID: stringField(m, "device_auth_id"),
		UserCode:     stringField(m, "user_code"),
		Interval:     interval,
	}, nil
}

func stringField(m map[string]any, key string) string {
	if s, ok := m[key].(string); ok {
		return s
	}
	return ""
}

// Interactive login stubs to satisfy build; actual interactive flows are implemented elsewhere in CLI.
func LoginBrowser(cfg OAuthProviderConfig) (*AuthCredential, error) {
	return nil, fmt.Errorf("interactive browser OAuth flow is not available in this build")
}

func LoginDeviceCode(cfg OAuthProviderConfig) (*AuthCredential, error) {
	return nil, fmt.Errorf("device code OAuth flow is not available in this build")
}
