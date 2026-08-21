package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/samkc-0/pamphlet-sync/internal/version"
)

// Version reports the commit and build time the running binary was built
// from, so a deploy can be confirmed without digging through CI logs.
func Version(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"commit":    version.CommitSHA,
		"buildTime": version.BuildTime,
	})
}
