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
	Name           string  `json:"name" binding:"required"`
	Service        string  `json:"service" binding:"required"`
	DisplayName    string  `json:"displayName"`
	URL            string  `json:"url" binding:"required"`
	Type           string  `json:"type"`
	AuthSecretName *string `json:"authSecretName,omitempty"`
	AuthSecretKey  *string `json:"authSecretKey,omitempty"`
	Namespace      string  `json:"namespace"`
}

type ConnectorResponse struct {
	Name           string   `json:"name"`
	Namespace      string   `json:"namespace"`
	Service        string   `json:"service"`
	DisplayName    string   `json:"displayName"`
	URL            string   `json:"url"`
	Type           string   `json:"type"`
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
		conn, err := k8s.CreateConnector(c.Request.Context(), ns, req.Name, req.Service, req.DisplayName, req.URL, connType, req.AuthSecretName, req.AuthSecretKey)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create connector: " + err.Error()})
			return
		}
		c.JSON(http.StatusCreated, connectorToResponse(conn, nil))
	}
}

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
				authHeader = "Bearer " + token
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
	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/list",
		"params":  map[string]interface{}{},
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", serverURL, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Handle SSE response (text/event-stream)
	contentType := resp.Header.Get("Content-Type")
	log.Printf("MCP tools/list response: status=%d content-type=%q url=%s", resp.StatusCode, contentType, serverURL)
	fullBody, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, fmt.Errorf("failed to read MCP response body: %w", readErr)
	}
	var rawBody []byte
	if strings.Contains(contentType, "text/event-stream") {
		// Extract the JSON payload from the first "data: {...}" SSE line
		log.Printf("MCP SSE raw body (first 512 bytes): %s", truncate(string(fullBody), 512))
		for _, line := range strings.Split(string(fullBody), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "data:") {
				rawBody = []byte(strings.TrimSpace(strings.TrimPrefix(line, "data:")))
				break
			}
		}
	} else {
		rawBody = fullBody
		log.Printf("MCP JSON raw body (first 512 bytes): %s", truncate(string(rawBody), 512))
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
		AttachedAgents: len(agentNames),
		AgentNames:     agentNames,
		CreatedAt:      conn.CreationTimestamp.UTC().Format(time.RFC3339),
	}
	if conn.Spec.AuthSecretKeyRef != nil {
		resp.AuthSecretName = conn.Spec.AuthSecretKeyRef.Name
		resp.AuthSecretKey = conn.Spec.AuthSecretKeyRef.Key
	}
	return resp
}
