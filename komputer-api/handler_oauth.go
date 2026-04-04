package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// oauthProvider holds static configuration for an OAuth 2.0 provider.
// Client credentials are per-connector (stored in K8s Secrets), not here.
type oauthProvider struct {
	AuthURL     string
	TokenURL    string
	Scopes      []string
	ExtraParams map[string]string
}

// oauthProviderRegistry maps service names to their OAuth provider config.
// Credentials are NOT stored here — they come from each connector's secret.
// MCP server URLs come from the UI connector templates, not here.
var oauthProviderRegistry = map[string]*oauthProvider{
	"google": {
		AuthURL:     "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL:    "https://oauth2.googleapis.com/token",
		Scopes:      []string{"https://www.googleapis.com/auth/gmail.modify", "https://www.googleapis.com/auth/calendar"},
		ExtraParams: map[string]string{"access_type": "offline", "prompt": "consent"},
	},
	"notion": {
		AuthURL:     "https://api.notion.com/v1/oauth/authorize",
		TokenURL:    "https://api.notion.com/v1/oauth/token",
		Scopes:      nil,
		ExtraParams: map[string]string{"owner": "user"},
	},
}

// resolveOAuthProvider resolves a service name (including aliases) to a provider.
func resolveOAuthProvider(service string) *oauthProvider {
	// Resolve aliases
	switch service {
	case "gmail", "google-calendar":
		service = "google"
	}
	return oauthProviderRegistry[service]
}

// oauthClientCredentials holds per-connector OAuth client credentials read from K8s Secret.
type oauthClientCredentials struct {
	ClientID     string
	ClientSecret string
}

// oauthPendingFlow stores state for an in-progress OAuth flow.
// All connector details are stashed here so the connector is only created on successful callback.
type oauthPendingFlow struct {
	ConnectorName     string
	Namespace         string
	Service           string
	DisplayName       string
	URL               string
	OAuthClientID     string
	OAuthClientSecret string
	CreatedAt         time.Time
}

// pendingOAuthFlows maps state string → *oauthPendingFlow.
var pendingOAuthFlows sync.Map

// generateOAuthState returns a cryptographically random hex state string.
func generateOAuthState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// oauthAuthorize handles POST /api/v1/oauth/authorize
// Accepts full connector details so the connector is only created after successful OAuth callback.
func oauthAuthorize(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Service           string `json:"service" binding:"required"`
			ConnectorName     string `json:"connector_name" binding:"required"`
			Namespace         string `json:"namespace"`
			DisplayName       string `json:"displayName"`
			URL               string `json:"url"`
			OAuthClientID     string `json:"oauthClientId" binding:"required"`
			OAuthClientSecret string `json:"oauthClientSecret" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
			return
		}
		if req.Namespace == "" {
			req.Namespace = resolveNamespace(c, k8s)
		}

		provider := resolveOAuthProvider(req.Service)
		if provider == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported service: " + req.Service})
			return
		}

		state, err := generateOAuthState()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate state"})
			return
		}

		pendingOAuthFlows.Store(state, &oauthPendingFlow{
			ConnectorName:     req.ConnectorName,
			Namespace:         req.Namespace,
			Service:           req.Service,
			DisplayName:       req.DisplayName,
			URL:               req.URL,
			OAuthClientID:     req.OAuthClientID,
			OAuthClientSecret: req.OAuthClientSecret,
			CreatedAt:         time.Now(),
		})

		callbackURL := resolveCallbackURL(c)

		params := url.Values{}
		params.Set("client_id", req.OAuthClientID)
		params.Set("redirect_uri", callbackURL)
		params.Set("response_type", "code")
		params.Set("state", state)
		if len(provider.Scopes) > 0 {
			params.Set("scope", strings.Join(provider.Scopes, " "))
		}
		for k, v := range provider.ExtraParams {
			params.Set(k, v)
		}

		authorizeURL := provider.AuthURL + "?" + params.Encode()
		log.Printf("OAuth authorize: service=%s connector=%s/%s callback=%s", req.Service, req.Namespace, req.ConnectorName, callbackURL)
		c.JSON(http.StatusOK, gin.H{"authorizeUrl": authorizeURL})
	}
}

// oauthCallback handles GET /api/v1/oauth/callback
// Query params: code, state, error (optional)
func oauthCallback(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		if errParam := c.Query("error"); errParam != "" {
			c.Data(http.StatusBadRequest, "text/html; charset=utf-8", []byte(oauthErrorHTML("OAuth error: "+errParam)))
			return
		}

		code := c.Query("code")
		state := c.Query("state")
		if code == "" || state == "" {
			c.Data(http.StatusBadRequest, "text/html; charset=utf-8", []byte(oauthErrorHTML("Missing code or state")))
			return
		}

		val, ok := pendingOAuthFlows.LoadAndDelete(state)
		if !ok {
			c.Data(http.StatusBadRequest, "text/html; charset=utf-8", []byte(oauthErrorHTML("Invalid or expired state")))
			return
		}
		flow := val.(*oauthPendingFlow)

		if time.Since(flow.CreatedAt) > 10*time.Minute {
			c.Data(http.StatusBadRequest, "text/html; charset=utf-8", []byte(oauthErrorHTML("OAuth flow expired")))
			return
		}

		provider := resolveOAuthProvider(flow.Service)
		if provider == nil {
			c.Data(http.StatusInternalServerError, "text/html; charset=utf-8", []byte(oauthErrorHTML("Unknown provider")))
			return
		}

		callbackURL := resolveCallbackURL(c)
		ctx := c.Request.Context()

		creds := oauthClientCredentials{
			ClientID:     flow.OAuthClientID,
			ClientSecret: flow.OAuthClientSecret,
		}

		tokenData, err := exchangeCodeForTokens(provider, creds, code, callbackURL)
		if err != nil {
			log.Printf("OAuth token exchange error: %v", err)
			c.Data(http.StatusInternalServerError, "text/html; charset=utf-8", []byte(oauthErrorHTML("Token exchange failed: "+err.Error())))
			return
		}

		// Determine canonical service name for storage (resolve alias).
		canonicalService := flow.Service
		if flow.Service == "gmail" || flow.Service == "google-calendar" {
			canonicalService = "google"
		}
		tokenData["service"] = canonicalService

		tokenJSON, err := json.Marshal(tokenData)
		if err != nil {
			c.Data(http.StatusInternalServerError, "text/html; charset=utf-8", []byte(oauthErrorHTML("Failed to serialize token")))
			return
		}

		// Create the secret with client credentials + OAuth token (all in one).
		secretName := flow.ConnectorName + "-oauth"
		secretKey := "oauth-token"

		if _, err := k8s.CreateManagedSecret(ctx, flow.Namespace, secretName, map[string]string{
			"client_id":     flow.OAuthClientID,
			"client_secret": flow.OAuthClientSecret,
			secretKey:       string(tokenJSON),
		}); err != nil {
			log.Printf("OAuth: failed to create secret %s/%s: %v", flow.Namespace, secretName, err)
			c.Data(http.StatusInternalServerError, "text/html; charset=utf-8", []byte(oauthErrorHTML("Failed to store token")))
			return
		}

		// Now create the connector CR with auth pointing at the secret.
		sn := secretName
		sk := secretKey
		conn, err := k8s.CreateConnector(ctx, flow.Namespace, flow.ConnectorName, flow.Service, flow.DisplayName, flow.URL, "remote", "oauth", &sn, &sk)
		if err != nil {
			log.Printf("OAuth: failed to create connector %s/%s: %v", flow.Namespace, flow.ConnectorName, err)
			c.Data(http.StatusInternalServerError, "text/html; charset=utf-8", []byte(oauthErrorHTML("Failed to create connector: "+err.Error())))
			return
		}

		// Set owner reference on secret → connector for garbage collection.
		k8s.SetSecretOwnerRef(ctx, flow.Namespace, secretName, conn.Name, string(conn.UID))

		log.Printf("OAuth success: service=%s connector=%s/%s secret=%s", flow.Service, flow.Namespace, flow.ConnectorName, secretName)
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(oauthSuccessHTML(flow.ConnectorName)))
	}
}

// oauthRefresh handles POST /api/v1/oauth/refresh
// JSON body: {"connector_name": "...", "namespace": "..."}
func oauthRefresh(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			ConnectorName string `json:"connector_name" binding:"required"`
			Namespace     string `json:"namespace"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
			return
		}
		if req.Namespace == "" {
			req.Namespace = resolveNamespace(c, k8s)
		}

		ctx := c.Request.Context()
		conn, err := k8s.GetConnector(ctx, req.Namespace, req.ConnectorName)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "connector not found"})
			return
		}
		if conn.Spec.AuthSecretKeyRef == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "connector has no auth secret"})
			return
		}

		secretName := conn.Spec.AuthSecretKeyRef.Name
		secretKey := conn.Spec.AuthSecretKeyRef.Key
		tokenData, err := k8s.GetOAuthTokenData(ctx, req.Namespace, secretName, secretKey)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read token: " + err.Error()})
			return
		}

		// Determine service from token data, fallback to connector spec.
		service, _ := tokenData["service"].(string)
		if service == "" {
			service = conn.Spec.Service
		}

		provider := resolveOAuthProvider(service)
		if provider == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported service: " + service})
			return
		}

		// Notion tokens don't expire — return the existing token.
		if service == "notion" {
			accessToken, _ := tokenData["access_token"].(string)
			c.JSON(http.StatusOK, gin.H{"access_token": accessToken})
			return
		}

		// Google: refresh using refresh_token.
		refreshToken, _ := tokenData["refresh_token"].(string)
		if refreshToken == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no refresh_token available"})
			return
		}

		// Read OAuth client credentials from connector's secret.
		creds, credErr := getConnectorOAuthCreds(ctx, k8s, req.Namespace, req.ConnectorName)
		if credErr != nil || creds.ClientID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "OAuth client credentials not found for connector"})
			return
		}

		newTokenData, err := refreshGoogleToken(provider, creds, refreshToken)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "token refresh failed: " + err.Error()})
			return
		}

		// Merge new fields into existing token data, preserving refresh_token and service.
		tokenData["access_token"] = newTokenData["access_token"]
		if ea, ok := newTokenData["expires_at"]; ok {
			tokenData["expires_at"] = ea
		}

		if err := k8s.UpdateOAuthTokenData(ctx, req.Namespace, secretName, secretKey, tokenData); err != nil {
			log.Printf("OAuth refresh: failed to update secret %s/%s: %v", req.Namespace, secretName, err)
			// Still return the new token even if secret update failed.
		}

		accessToken, _ := tokenData["access_token"].(string)
		expiresAt, _ := tokenData["expires_at"].(float64)
		c.JSON(http.StatusOK, gin.H{"access_token": accessToken, "expires_at": expiresAt})
	}
}

// exchangeCodeForTokens exchanges an authorization code for OAuth tokens.
func exchangeCodeForTokens(provider *oauthProvider, creds oauthClientCredentials, code, redirectURI string) (map[string]interface{}, error) {
	var (
		resp *http.Response
		err  error
	)

	client := &http.Client{Timeout: 15 * time.Second}

	if strings.Contains(provider.TokenURL, "notion.com") {
		// Notion uses Basic auth + JSON body.
		body, _ := json.Marshal(map[string]string{
			"code":         code,
			"grant_type":   "authorization_code",
			"redirect_uri": redirectURI,
		})
		req, reqErr := http.NewRequest("POST", provider.TokenURL, strings.NewReader(string(body)))
		if reqErr != nil {
			return nil, reqErr
		}
		req.Header.Set("Content-Type", "application/json")
		basicAuth := base64.StdEncoding.EncodeToString([]byte(creds.ClientID + ":" + creds.ClientSecret))
		req.Header.Set("Authorization", "Basic "+basicAuth)
		resp, err = client.Do(req)
	} else {
		// Standard form-encoded POST (Google, etc.).
		form := url.Values{}
		form.Set("code", code)
		form.Set("client_id", creds.ClientID)
		form.Set("client_secret", creds.ClientSecret)
		form.Set("redirect_uri", redirectURI)
		form.Set("grant_type", "authorization_code")
		resp, err = client.PostForm(provider.TokenURL, form)
	}

	if err != nil {
		return nil, fmt.Errorf("token exchange request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange returned %d: %s", resp.StatusCode, truncate(string(body), 256))
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	result := map[string]interface{}{
		"access_token":  raw["access_token"],
		"refresh_token": raw["refresh_token"],
		"token_type":    "Bearer",
	}
	if expiresIn, ok := raw["expires_in"].(float64); ok && expiresIn > 0 {
		result["expires_at"] = float64(time.Now().Unix()) + expiresIn
	}

	return result, nil
}

// refreshGoogleToken uses the refresh_token to obtain a new access_token from Google.
func refreshGoogleToken(provider *oauthProvider, creds oauthClientCredentials, refreshToken string) (map[string]interface{}, error) {
	form := url.Values{}
	form.Set("refresh_token", refreshToken)
	form.Set("client_id", creds.ClientID)
	form.Set("client_secret", creds.ClientSecret)
	form.Set("grant_type", "refresh_token")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.PostForm(provider.TokenURL, form)
	if err != nil {
		return nil, fmt.Errorf("refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("refresh returned %d: %s", resp.StatusCode, truncate(string(body), 256))
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse refresh response: %w", err)
	}

	result := map[string]interface{}{
		"access_token": raw["access_token"],
	}
	if expiresIn, ok := raw["expires_in"].(float64); ok && expiresIn > 0 {
		result["expires_at"] = float64(time.Now().Unix()) + expiresIn
	}
	return result, nil
}

// getConnectorOAuthCreds reads OAuth client_id and client_secret from a connector's secret.
func getConnectorOAuthCreds(ctx context.Context, k8s *K8sClient, ns, connectorName string) (oauthClientCredentials, error) {
	secretName := connectorName + "-oauth"
	clientID, err := k8s.GetSecretValue(ctx, ns, secretName, "client_id")
	if err != nil {
		return oauthClientCredentials{}, err
	}
	clientSecret, _ := k8s.GetSecretValue(ctx, ns, secretName, "client_secret")
	return oauthClientCredentials{ClientID: clientID, ClientSecret: clientSecret}, nil
}

// resolveCallbackURL returns the OAuth callback URL.
// Uses OAUTH_CALLBACK_URL env var if set, otherwise derives from the incoming request.
func resolveCallbackURL(c *gin.Context) string {
	if env := os.Getenv("OAUTH_CALLBACK_URL"); env != "" {
		return env
	}
	scheme := "https"
	if c.Request.TLS == nil {
		if fwd := c.GetHeader("X-Forwarded-Proto"); fwd != "" {
			scheme = fwd
		} else {
			scheme = "http"
		}
	}
	return scheme + "://" + c.Request.Host + "/api/v1/oauth/callback"
}

// oauthSuccessHTML returns an HTML page that signals success to the popup opener.
// Uses both postMessage (direct) and localStorage (cross-origin fallback).
func oauthSuccessHTML(connectorName string) string {
	return fmt.Sprintf(`<html><body><script>
  try { localStorage.setItem("oauth-success", "%s:" + Date.now()); } catch(e) {}
  if (window.opener) {
    window.opener.postMessage({type:"oauth-success",connector:"%s"},"*");
  }
  setTimeout(function() { window.close(); }, 500);
</script><p>Connected! You can close this window.</p></body></html>`, connectorName, connectorName)
}

// oauthErrorHTML returns an HTML page that signals failure to the popup opener.
func oauthErrorHTML(msg string) string {
	return fmt.Sprintf(`<html><body><script>
  window.opener && window.opener.postMessage({type:"oauth-error",error:"%s"},"*");
</script><p>Error: %s</p></body></html>`, msg, msg)
}
