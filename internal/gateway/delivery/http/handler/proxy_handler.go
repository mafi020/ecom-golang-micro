package handler

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gin-gonic/gin"
)

type ProxyHandler struct {
	target *url.URL
	proxy  *httputil.ReverseProxy
}

func NewProxyHandler(targetURL string) (*ProxyHandler, error) {
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return nil, err
	}

	proxy := httputil.NewSingleHostReverseProxy(parsedURL)

	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = parsedURL.Host
	}

	return &ProxyHandler{
		target: parsedURL,
		proxy:  proxy,
	}, nil
}

// ProxyRequest intercepts the context, copies variables, and forces them into the active outbound request stream
func (ph *ProxyHandler) ProxyRequest() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract values populated by your Gateway AuthMiddleware
		if userID, exists := c.Get("user_id"); exists {
			c.Request.Header.Set("X-User-ID", fmt.Sprintf("%v", userID))
		}
		if role, exists := c.Get("role"); exists {
			c.Request.Header.Set("X-User-Role", fmt.Sprintf("%v", role))
		}

		// Stream the mutated request over to the Monolith network socket
		ph.proxy.ServeHTTP(c.Writer, c.Request)
	}
}
