package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const agentFilesDir = "/files"

// downloadAgentFile proxies a file download from the agent pod's /files directory.
// Primary: HTTP GET to the agent's /download endpoint.
// Fallback: kubectl exec cat (for local dev when pod IP is unreachable).
// GET /api/v1/agents/:name/download/*filepath
func downloadAgentFile(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		agentName := c.Param("name")
		reqPath := c.Param("filepath")
		ns := resolveNamespace(c, k8s)

		// Prevent directory traversal.
		if strings.Contains(reqPath, "..") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file path"})
			return
		}

		agent, err := k8s.GetAgent(c.Request.Context(), ns, agentName)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
			return
		}
		podName := agent.Status.PodName
		if podName == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "agent has no running pod"})
			return
		}

		filePath := strings.TrimPrefix(reqPath, "/")
		filename := filepath.Base(filePath)
		fullPath := agentFilesDir + "/" + filePath

		// Try HTTP proxy first (skipped in LOCAL mode where in-cluster DNS isn't available).
		if os.Getenv("LOCAL") != "true" {
			agentURL := fmt.Sprintf("%s/download/%s", agentHTTPBase(ns, agentName), filePath)
			data, contentType, proxyFilename, proxyErr := proxyFileFromAgent(c.Request.Context(), agentURL)
			if proxyErr == nil {
				if proxyFilename != "" {
					filename = proxyFilename
				}
				c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
				c.Data(http.StatusOK, contentType, data)
				return
			}
			Logger.Warnw("file proxy failed, falling back to exec", "namespace", ns, "agent_name", agentName, "error", proxyErr)
		}

		// Fallback: kubectl exec cat (binary-safe, works locally).
		// For squad pods, target the per-agent container by name.
		data, err := k8s.execInContainerWithOutput(c.Request.Context(), ns, podName, agentName, "cat", fullPath)
		if err != nil {
			Logger.Errorw("file exec fallback failed", "namespace", ns, "agent_name", agentName, "path", fullPath, "error", err)
			c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
			return
		}

		c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
		c.Data(http.StatusOK, guessContentType(filename), data)
	}
}

// proxyFileFromAgent makes a GET request to the agent pod and returns the file bytes.
func proxyFileFromAgent(ctx context.Context, agentURL string) (data []byte, contentType string, filename string, err error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(timeoutCtx, http.MethodGet, agentURL, nil)
	if err != nil {
		return nil, "", "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, "", "", fmt.Errorf("agent returned %d: %s", resp.StatusCode, string(body))
	}

	data, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", "", err
	}

	contentType = resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Extract filename from Content-Disposition if present.
	if cd := resp.Header.Get("Content-Disposition"); cd != "" {
		if idx := strings.Index(cd, "filename="); idx != -1 {
			filename = strings.Trim(cd[idx+9:], `"' `)
		}
	}

	return data, contentType, filename, nil
}

func guessContentType(filename string) string {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".json":
		return "application/json"
	case ".txt", ".md", ".log", ".csv":
		return "text/plain; charset=utf-8"
	case ".html":
		return "text/html; charset=utf-8"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".svg":
		return "image/svg+xml"
	case ".pdf":
		return "application/pdf"
	case ".excalidraw":
		return "application/json"
	default:
		return "application/octet-stream"
	}
}
