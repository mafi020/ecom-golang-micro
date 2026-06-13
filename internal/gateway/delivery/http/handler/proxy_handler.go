package handler

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

type ProxyHandler struct {
	target *url.URL
	proxy  *httputil.ReverseProxy
	prefix string // 🚀 FIXED: Explicitly track the prefix to trim
}

func NewProxyHandler(targetURL string, prefix string) (*ProxyHandler, error) {
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
		prefix: prefix,
	}, nil
}

func (ph *ProxyHandler) ProxyRequest() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Clean out malicious spoofed headers
		c.Request.Header.Del("X-User-ID")
		c.Request.Header.Del("X-User-Role")

		// Inject genuine middleware authenticated parameters
		if userID, exists := c.Get("user_id"); exists {
			c.Request.Header.Set("X-User-ID", fmt.Sprintf("%v", userID))
		}
		if role, exists := c.Get("role"); exists {
			c.Request.Header.Set("X-User-Role", fmt.Sprintf("%v", role))
		}

		// 🚀 FIXED: Perform predictable, error-free path trimming using the explicit prefix
		if ph.prefix != "" && strings.HasPrefix(c.Request.URL.Path, ph.prefix) {
			c.Request.URL.Path = strings.TrimPrefix(c.Request.URL.Path, ph.prefix)
		}

		// Ensure the path retains a safe leading slash character mapping
		if c.Request.URL.Path == "" || !strings.HasPrefix(c.Request.URL.Path, "/") {
			c.Request.URL.Path = "/" + c.Request.URL.Path
		}

		// Forward cleanly over the proxy network bridge
		ph.proxy.ServeHTTP(c.Writer, c.Request)
	}
}
