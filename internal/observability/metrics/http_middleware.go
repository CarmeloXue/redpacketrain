package metrics

import (
	"time"

	"github.com/gin-gonic/gin"
)

// GinMiddleware records latency metrics for each HTTP request.
func GinMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		route := c.FullPath()
		if route == "" {
			route = c.Request.URL.Path
		}
		ObserveHTTPRequest(c.Request.Method, route, c.Writer.Status(), time.Since(start))
	}
}
