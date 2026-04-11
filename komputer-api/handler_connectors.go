package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

type CreateConnectorRequest struct {
	Name              string  `json:"name" binding:"required"`
	Service           string  `json:"service" binding:"required"`
	DisplayName       string  `json:"displayName"`
	URL               string  `json:"url" binding:"required"`
	Type              string  `json:"type"`
	AuthType          string  `json:"authType,omitempty"`          // "token" or "oauth"
	AuthSecretName    *string `json:"authSecretName,omitempty"`
	AuthSecretKey     *string `json:"authSecretKey,omitempty"`
	OAuthClientID     string  `json:"oauthClientId,omitempty"`     // OAuth client ID (stored in secret)
	OAuthClientSecret string  `json:"oauthClientSecret,omitempty"` // OAuth client secret (stored in secret)
	Namespace         string  `json:"namespace"`
}

type ConnectorResponse struct {
	Name           string   `json:"name"`
	Namespace      string   `json:"namespace"`
	Service        string   `json:"service"`
	DisplayName    string   `json:"displayName"`
	URL            string   `json:"url"`
	Type           string   `json:"type"`
	AuthType       string   `json:"authType,omitempty"`
	OAuthStatus    string   `json:"oauthStatus,omitempty"` // "pending", "connected", ""
	AuthSecretName string   `json:"authSecretName,omitempty"`
	AuthSecretKey  string   `json:"authSecretKey,omitempty"`
	AttachedAgents int      `json:"attachedAgents"`
	AgentNames     []string `json:"agentNames,omitempty"`
	CreatedAt      string   `json:"createdAt"`
}

type mcpTool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// createConnector creates a new MCP connector resource.
// @ID createConnector
// @Summary Create connector
// @Description Creates a new KomputerConnector CR pointing to an MCP server that can be attached to agents.
// @Tags connectors
// @Accept json
// @Produce json
// @Param request body CreateConnectorRequest true "Connector creation request"
// @Success 201 {object} ConnectorResponse "Connector created"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 500 {object} map[string]string "Internal error"
// @Router /connectors [post]
func createConnector(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateConnectorRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
			return
		}
		ns := req.Namespace
		if ns == "" {
			ns = resolveNamespace(c, k8s)
		}
		connType := req.Type
		if connType == "" {
			connType = "remote"
		}
		conn, err := k8s.CreateConnector(c.Request.Context(), ns, req.Name, req.Service, req.DisplayName, req.URL, connType, req.AuthType, req.AuthSecretName, req.AuthSecretKey)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create connector: " + err.Error()})
			return
		}
		c.JSON(http.StatusCreated, connectorToResponse(conn, nil))
	}
}

// listConnectors returns all connectors in a namespace.
// @ID listConnectors
// @Summary List connectors
// @Description Returns all connectors with attached agent counts in the specified namespace.
// @Tags connectors
// @Produce json
// @Param namespace query string false "Kubernetes namespace"
// @Success 200 {object} map[string]interface{} "List of connectors"
// @Failure 500 {object} map[string]string "Internal error"
// @Router /connectors [get]
func listConnectors(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		ns := c.Query("namespace")
		connectors, err := k8s.ListConnectors(c.Request.Context(), ns)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list connectors: " + err.Error()})
			return
		}
		connUsage := make(map[string][]string)
		agents, _ := k8s.ListAgents(c.Request.Context(), "")
		for _, a := range agents {
			for _, connRef := range a.Spec.Connectors {
				key := a.Namespace + "/" + connRef
				connUsage[key] = append(connUsage[key], a.Name)
			}
		}
		resp := make([]ConnectorResponse, 0, len(connectors))
		for _, conn := range connectors {
			agentNames := connUsage[conn.Namespace+"/"+conn.Name]
			resp = append(resp, connectorToResponse(&conn, agentNames))
		}
		c.JSON(http.StatusOK, gin.H{"connectors": resp})
	}
}

// getConnector returns details for a single connector.
// @ID getConnector
// @Summary Get connector details
// @Description Returns the URL, service, type, and auth config for a single connector.
// @Tags connectors
// @Produce json
// @Param name path string true "Connector name"
// @Param namespace query string false "Kubernetes namespace"
// @Success 200 {object} ConnectorResponse "Connector details"
// @Failure 404 {object} map[string]string "Connector not found"
// @Failure 500 {object} map[string]string "Internal error"
// @Router /connectors/{name} [get]
func getConnector(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		ns := resolveNamespace(c, k8s)
		conn, err := k8s.GetConnector(c.Request.Context(), ns, name)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "connector not found"})
			return
		}
		c.JSON(http.StatusOK, connectorToResponse(conn, nil))
	}
}

// listConnectorTools fetches the available tools from the connector's MCP server.
// @ID listConnectorTools
// @Summary List connector tools
// @Description Calls the MCP server's tools/list endpoint and returns the available tools.
// @Tags connectors
// @Produce json
// @Param name path string true "Connector name"
// @Param namespace query string false "Kubernetes namespace"
// @Success 200 {object} map[string]interface{} "List of MCP tools"
// @Failure 404 {object} map[string]string "Connector not found"
// @Failure 502 {object} map[string]string "Failed to reach MCP server"
// @Failure 500 {object} map[string]string "Internal error"
// @Router /connectors/{name}/tools [get]
func listConnectorTools(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		ns := resolveNamespace(c, k8s)
		conn, err := k8s.GetConnector(c.Request.Context(), ns, name)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "connector not found"})
			return
		}
		if conn.Spec.URL == "" {
			c.JSON(http.StatusOK, gin.H{"tools": []interface{}{}})
			return
		}

		// Resolve auth token from secret if present.
		// Use conn.Namespace (the connector's actual namespace) — not the resolved query param —
		// so the secret is always looked up where it was created.
		authHeader := ""
		if conn.Spec.AuthSecretKeyRef != nil {
			token, err := k8s.GetSecretValue(c.Request.Context(), conn.Namespace, conn.Spec.AuthSecretKeyRef.Name, conn.Spec.AuthSecretKeyRef.Key)
			if err == nil && token != "" {
				if conn.Spec.AuthType == "oauth" {
					// Secret value is a JSON blob: {"access_token": "...", ...}
					var oauthData struct {
						AccessToken string `json:"access_token"`
					}
					if jsonErr := json.Unmarshal([]byte(token), &oauthData); jsonErr == nil && oauthData.AccessToken != "" {
						authHeader = "Bearer " + oauthData.AccessToken
					}
				} else {
					authHeader = "Bearer " + token
				}
			}
		}

		log.Printf("fetching MCP tools for connector %s/%s url=%s auth=%v", conn.Namespace, conn.Name, conn.Spec.URL, authHeader != "")
		tools, err := fetchMCPTools(conn.Spec.URL, authHeader)
		if err != nil {
			log.Printf("error fetching MCP tools for connector %s/%s: %v", conn.Namespace, conn.Name, err)
			c.JSON(http.StatusBadGateway, gin.H{"error": "failed to fetch tools from MCP server: " + err.Error()})
			return
		}
		log.Printf("fetched %d tools for connector %s/%s", len(tools), conn.Namespace, conn.Name)
		c.JSON(http.StatusOK, gin.H{"tools": tools})
	}
}

func fetchMCPTools(serverURL, authHeader string) ([]mcpTool, error) {
	client := &http.Client{Timeout: 15 * time.Second}

	// Helper to make an MCP JSON-RPC request and return the raw JSON body.
	mcpRequest := func(method string, id int, sessionID string) ([]byte, string, http.Header, error) {
		payload, _ := json.Marshal(map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      id,
			"method":  method,
			"params":  map[string]interface{}{},
		})
		if method == "initialize" {
			payload, _ = json.Marshal(map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      id,
				"method":  method,
				"params": map[string]interface{}{
					"protocolVersion": "2025-03-26",
					"capabilities":    map[string]interface{}{},
					"clientInfo":      map[string]interface{}{"name": "komputer", "version": "1.0"},
				},
			})
		}
		req, err := http.NewRequest("POST", serverURL, strings.NewReader(string(payload)))
		if err != nil {
			return nil, "", nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		if authHeader != "" {
			req.Header.Set("Authorization", authHeader)
		}
		if sessionID != "" {
			req.Header.Set("Mcp-Session-Id", sessionID)
		}

		resp, err := client.Do(req)
		if err != nil {
			return nil, "", nil, err
		}
		defer resp.Body.Close()

		contentType := resp.Header.Get("Content-Type")
		fullBody, _ := io.ReadAll(resp.Body)

		// Extract JSON from SSE if needed.
		var rawBody []byte
		if strings.Contains(contentType, "text/event-stream") {
			for _, line := range strings.Split(string(fullBody), "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "data:") {
					rawBody = []byte(strings.TrimSpace(strings.TrimPrefix(line, "data:")))
					break
				}
			}
		} else {
			rawBody = fullBody
		}
		return rawBody, contentType, resp.Header, nil
	}

	// Step 1: Try tools/list directly (works for servers that don't require sessions).
	rawBody, contentType, respHeaders, err := mcpRequest("tools/list", 1, "")
	if err != nil {
		return nil, err
	}
	log.Printf("MCP tools/list response: content-type=%q url=%s", contentType, serverURL)

	// Check if server requires a session (Mcp-Session-Id header).
	needsSession := false
	if rawBody != nil {
		var errCheck struct {
			Error *struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal(rawBody, &errCheck) == nil && errCheck.Error != nil &&
			strings.Contains(errCheck.Error.Message, "Session-Id") {
			needsSession = true
		}
	}

	if needsSession {
		// Step 2: Initialize to get a session ID.
		initBody, _, initHeaders, initErr := mcpRequest("initialize", 1, "")
		if initErr != nil {
			return nil, fmt.Errorf("MCP initialize failed: %w", initErr)
		}
		sessionID := initHeaders.Get("Mcp-Session-Id")
		if sessionID == "" {
			return nil, fmt.Errorf("MCP server did not return Mcp-Session-Id after initialize")
		}
		log.Printf("MCP session established: %s (init response: %s)", sessionID, truncate(string(initBody), 200))
		_ = respHeaders // unused now

		// Step 3: tools/list with session.
		rawBody, _, _, err = mcpRequest("tools/list", 2, sessionID)
		if err != nil {
			return nil, fmt.Errorf("MCP tools/list with session failed: %w", err)
		}
	}

	var rpcResp struct {
		Result struct {
			Tools []struct {
				Name        string `json:"name"`
				Description string `json:"description"`
			} `json:"tools"`
		} `json:"result"`
		Error json.RawMessage `json:"error"`
	}
	if err := json.Unmarshal(rawBody, &rpcResp); err != nil {
		log.Printf("MCP tools/list unmarshal error: %v — raw: %s", err, truncate(string(rawBody), 256))
		return nil, fmt.Errorf("invalid MCP response: %w", err)
	}
	if len(rpcResp.Error) > 0 && string(rpcResp.Error) != "null" {
		// Error may be a string or an object with a "message" field.
		var errMsg string
		var errObj struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(rpcResp.Error, &errObj) == nil && errObj.Message != "" {
			errMsg = errObj.Message
		} else {
			_ = json.Unmarshal(rpcResp.Error, &errMsg)
		}
		log.Printf("MCP tools/list RPC error: %s", errMsg)
		return nil, fmt.Errorf("MCP error: %s", errMsg)
	}

	tools := make([]mcpTool, 0, len(rpcResp.Result.Tools))
	for _, t := range rpcResp.Result.Tools {
		tools = append(tools, mcpTool{Name: t.Name, Description: t.Description})
	}
	return tools, nil
}

// deleteConnector deletes a connector by name.
// @ID deleteConnector
// @Summary Delete connector
// @Description Deletes the connector CR.
// @Tags connectors
// @Produce json
// @Param name path string true "Connector name"
// @Param namespace query string false "Kubernetes namespace"
// @Success 200 {object} map[string]string "Connector deleted"
// @Failure 500 {object} map[string]string "Internal error"
// @Router /connectors/{name} [delete]
func deleteConnector(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		ns := resolveNamespace(c, k8s)
		if err := k8s.DeleteConnector(c.Request.Context(), ns, name); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete connector: " + err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "deleted"})
	}
}

func connectorToResponse(conn *komputerv1alpha1.KomputerConnector, agentNames []string) ConnectorResponse {
	resp := ConnectorResponse{
		Name:           conn.Name,
		Namespace:      conn.Namespace,
		Service:        conn.Spec.Service,
		DisplayName:    conn.Spec.DisplayName,
		URL:            conn.Spec.URL,
		Type:           conn.Spec.Type,
		AuthType:       conn.Spec.AuthType,
		AttachedAgents: len(agentNames),
		AgentNames:     agentNames,
		CreatedAt:      conn.CreationTimestamp.UTC().Format(time.RFC3339),
	}
	if conn.Spec.AuthSecretKeyRef != nil {
		resp.AuthSecretName = conn.Spec.AuthSecretKeyRef.Name
		resp.AuthSecretKey = conn.Spec.AuthSecretKeyRef.Key
	}
	if conn.Spec.AuthType == "oauth" {
		if conn.Spec.AuthSecretKeyRef != nil {
			resp.OAuthStatus = "connected"
		} else {
			resp.OAuthStatus = "pending"
		}
	}
	return resp
}
