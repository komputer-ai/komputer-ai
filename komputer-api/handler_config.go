package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// S3ConfigResponse says whether the S3 session store is on. It never carries
// credentials.
type S3ConfigResponse struct {
	Enabled bool   `json:"enabled"`
	Bucket  string `json:"bucket,omitempty"`
}

// ConfigResponse is the platform configuration clients need to adapt their UI.
type ConfigResponse struct {
	S3 S3ConfigResponse `json:"s3"`
}

// getConfig returns the platform configuration clients need: today, whether the
// S3 session store is on (which changes what archived and ephemeral agents can do).
// @ID getConfig
// @Summary Get platform configuration
// @Description Returns platform settings that change what clients can offer, such as whether the S3 session store is enabled. Never returns credentials.
// @Tags config
// @Produce json
// @Success 200 {object} ConfigResponse "Platform configuration"
// @Router /config [get]
func getConfig(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		resp := ConfigResponse{}
		if s3 := k8s.S3Config(c.Request.Context()); s3.IsEnabled() {
			resp.S3 = S3ConfigResponse{Enabled: true, Bucket: s3.Bucket}
		}
		c.JSON(http.StatusOK, resp)
	}
}
