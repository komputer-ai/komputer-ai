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

// oauthDiscovery holds the fields we care about from an OAuth 2.0 authorization server metadata document.
// See RFC 8414: https://datatracker.ietf.org/doc/html/rfc8414
type oauthDiscovery struct {
	Issuer                string   `json:"issuer"`
	AuthorizationEndpoint string   `json:"authorization_endpoint"`
	TokenEndpoint         string   `json:"token_endpoint"`
	RegistrationEndpoint  string   `json:"registration_endpoint"`
	CodeChallengeMethods  []string `json:"code_challenge_methods_supported"`
}

// discoveryCache caches discovered OAuth metadata keyed by MCP URL to avoid repeated fetches.
var discoveryCache sync.Map // mcpURL → *oauthDiscovery

// discoverOAuth fetches OAuth 2.0 authorization server metadata from the MCP server's origin.
// It tries {scheme}://{host}/.well-known/oauth-authorization-server and caches successful results.
func discoverOAuth(mcpURL string) (*oauthDiscovery, error) {
	if cached, ok := discoveryCache.Load(mcpURL); ok {
		return cached.(*oauthDiscovery), nil
	}

	base := strings.TrimRight(mcpURL, "/")
	u, err := url.Parse(base)
	if err != nil {
		return nil, fmt.Errorf("invalid MCP URL %q: %w", mcpURL, err)
	}
	wellKnownURL := u.Scheme + "://" + u.Host + "/.well-known/oauth-authorization-server"

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(wellKnownURL)
	if err != nil {
		return nil, fmt.Errorf("discovery request to %s failed: %w", wellKnownURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("discovery returned %d for %s", resp.StatusCode, wellKnownURL)
	}

	var disc oauthDiscovery
	if err := json.NewDecoder(resp.Body).Decode(&disc); err != nil {
		return nil, fmt.Errorf("failed to decode discovery response: %w", err)
	}

	discoveryCache.Store(mcpURL, &disc)
	return &disc, nil
}

// oauthQuirks holds provider-specific behavior that can't be auto-discovered.
type oauthQuirks struct {
	Scopes       []string          // scopes to request (empty = provider default)
	ExtraParams  map[string]string // extra query params on the authorize URL
	UseBasicAuth bool              // use HTTP Basic auth for token exchange (Notion)
}

// oauthQuirksMap maps canonical service names to their quirks.
// Aliases (gmail, google-calendar) are resolved in getQuirks.
var oauthQuirksMap = map[string]*oauthQuirks{
	"google": {
		Scopes:      []string{"https://www.googleapis.com/auth/gmail.modify", "https://www.googleapis.com/auth/calendar"},
		ExtraParams: map[string]string{"access_type": "offline", "prompt": "consent"},
	},
}

// getQuirks resolves aliases and returns the quirks for a service.
// Returns an empty (zero-value) quirks struct for unknown services — standard OAuth behavior.
func getQuirks(service string) *oauthQuirks {
	switch service {
	case "gmail", "google-calendar":
		service = "google"
	}
	if q, ok := oauthQuirksMap[service]; ok {
		return q
	}
	return &oauthQuirks{}
}

// registerOAuthClient performs dynamic client registration (RFC 7591) with an MCP OAuth server.
func registerOAuthClient(registrationEndpoint, clientName, redirectURI string) (clientID, clientSecret string, err error) {
	body, _ := json.Marshal(map[string]interface{}{
		"client_name":                  clientName,
		"redirect_uris":               []string{redirectURI},
		"grant_types":                  []string{"authorization_code", "refresh_token"},
		"response_types":              []string{"code"},
		"token_endpoint_auth_method":  "client_secret_post",
	})

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(registrationEndpoint, "application/json", strings.NewReader(string(body)))
	if err != nil {
		return "", "", fmt.Errorf("registration request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", "", fmt.Errorf("registration returned %d: %s", resp.StatusCode, truncate(string(respBody), 256))
	}

	var result struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", "", fmt.Errorf("failed to parse registration response: %w", err)
	}
	if result.ClientID == "" {
		return "", "", fmt.Errorf("registration returned empty client_id")
	}

	log.Printf("OAuth dynamic registration: endpoint=%s client_id=%s", registrationEndpoint, result.ClientID)
	return result.ClientID, result.ClientSecret, nil
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
			OAuthClientID     string `json:"oauthClientId"`
			OAuthClientSecret string `json:"oauthClientSecret"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
			return
		}
		if req.Namespace == "" {
			req.Namespace = resolveNamespace(c, k8s)
		}

		// Resolve the MCP URL from the connector template.
		tmpl := getConnectorTemplate(req.Service)
		if tmpl == nil || tmpl.URL == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "service not configured for OAuth: " + req.Service})
			return
		}

		// Auto-discover OAuth endpoints from the MCP server's well-known metadata.
		disc, err := discoverOAuth(tmpl.URL)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "OAuth discovery failed: " + err.Error()})
			return
		}

		callbackURL := resolveCallbackURL(c)

		// If no client credentials provided and server supports dynamic registration, auto-register.
		if req.OAuthClientID == "" && disc.RegistrationEndpoint != "" {
			clientID, clientSecret, regErr := registerOAuthClient(disc.RegistrationEndpoint, "komputer-"+req.ConnectorName, callbackURL)
			if regErr != nil {
				c.JSON(http.StatusBadGateway, gin.H{"error": "OAuth client registration failed: " + regErr.Error()})
				return
			}
			req.OAuthClientID = clientID
			req.OAuthClientSecret = clientSecret
		}
		if req.OAuthClientID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "OAuth client credentials required — this provider does not support auto-registration"})
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

		quirks := getQuirks(req.Service)

		params := url.Values{}
		params.Set("client_id", req.OAuthClientID)
		params.Set("redirect_uri", callbackURL)
		params.Set("response_type", "code")
		params.Set("state", state)
		if len(quirks.Scopes) > 0 {
			params.Set("scope", strings.Join(quirks.Scopes, " "))
		}
		for k, v := range quirks.ExtraParams {
			params.Set(k, v)
		}

		authorizeURL := disc.AuthorizationEndpoint + "?" + params.Encode()
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

		// Look up template to get the canonical MCP URL for discovery.
		tmpl := getConnectorTemplate(flow.Service)
		if tmpl == nil || tmpl.URL == "" {
			c.Data(http.StatusInternalServerError, "text/html; charset=utf-8", []byte(oauthErrorHTML("Unknown provider")))
			return
		}

		disc, err := discoverOAuth(tmpl.URL)
		if err != nil {
			c.Data(http.StatusBadGateway, "text/html; charset=utf-8", []byte(oauthErrorHTML("OAuth discovery failed: "+err.Error())))
			return
		}

		callbackURL := resolveCallbackURL(c)
		ctx := c.Request.Context()

		creds := oauthClientCredentials{
			ClientID:     flow.OAuthClientID,
			ClientSecret: flow.OAuthClientSecret,
		}

		quirks := getQuirks(flow.Service)
		tokenData, err := exchangeCodeForTokens(disc.TokenEndpoint, quirks, creds, code, callbackURL)
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

		quirks := getQuirks(service)

		// Notion tokens don't expire — return the existing token.
		if quirks.UseBasicAuth {
			accessToken, _ := tokenData["access_token"].(string)
			c.JSON(http.StatusOK, gin.H{"access_token": accessToken})
			return
		}

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

		// Look up MCP URL from template and auto-discover token endpoint.
		tmpl := getConnectorTemplate(service)
		if tmpl == nil || tmpl.URL == "" {
			// Fall back to connector's stored URL if template unavailable.
			tmpl = &ConnectorTemplateResponse{URL: conn.Spec.URL}
		}
		disc, err := discoverOAuth(tmpl.URL)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "OAuth discovery failed: " + err.Error()})
			return
		}

		newTokenData, err := refreshOAuthToken(disc.TokenEndpoint, quirks, creds, refreshToken)
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
// It uses Basic auth + JSON body if quirks.UseBasicAuth is set, otherwise standard form-encoded POST.
func exchangeCodeForTokens(tokenURL string, quirks *oauthQuirks, creds oauthClientCredentials, code, redirectURI string) (map[string]interface{}, error) {
	var (
		resp *http.Response
		err  error
	)

	client := &http.Client{Timeout: 15 * time.Second}

	if quirks.UseBasicAuth {
		// Basic auth + JSON body (e.g. Notion).
		body, _ := json.Marshal(map[string]string{
			"code":         code,
			"grant_type":   "authorization_code",
			"redirect_uri": redirectURI,
		})
		req, reqErr := http.NewRequest("POST", tokenURL, strings.NewReader(string(body)))
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
		resp, err = client.PostForm(tokenURL, form)
	}

	if err != nil {
		return nil, fmt.Errorf("token exchange request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange returned %d: %s", resp.StatusCode, truncate(string(body), 256))
	}

	log.Printf("OAuth token exchange raw response: %s", truncate(string(body), 512))

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

// refreshOAuthToken uses a refresh_token to obtain a new access_token.
func refreshOAuthToken(tokenURL string, quirks *oauthQuirks, creds oauthClientCredentials, refreshToken string) (map[string]interface{}, error) {
	form := url.Values{}
	form.Set("refresh_token", refreshToken)
	form.Set("client_id", creds.ClientID)
	form.Set("client_secret", creds.ClientSecret)
	form.Set("grant_type", "refresh_token")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.PostForm(tokenURL, form)
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
