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
	prefix string
}

func NewProxyHandler(targetURL string, prefix string) (*ProxyHandler, error) {
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return nil, err
	}
	return &ProxyHandler{target: parsedURL, prefix: prefix}, nil
}

func (ph *ProxyHandler) ProxyRequest() gin.HandlerFunc {
	return func(c *gin.Context) {
		req := c.Request.Clone(c.Request.Context())
		req.URL.Scheme = ph.target.Scheme
		req.URL.Host = ph.target.Host
		req.Host = ph.target.Host

		req.Header.Del("X-User-ID")
		req.Header.Del("X-User-Role")

		if userID, exists := c.Get("user_id"); exists {
			req.Header.Set("X-User-ID", fmt.Sprintf("%v", userID))
		}
		if role, exists := c.Get("role"); exists {
			req.Header.Set("X-User-Role", fmt.Sprintf("%v", role))
		}

		if ph.prefix != "" && strings.HasPrefix(req.URL.Path, ph.prefix) {
			req.URL.Path = strings.TrimPrefix(req.URL.Path, ph.prefix)
		}
		if req.URL.Path == "" || !strings.HasPrefix(req.URL.Path, "/") {
			req.URL.Path = "/" + req.URL.Path
		}

		transportProxy := &httputil.ReverseProxy{
			Director: func(outReq *http.Request) {
				outReq.URL.RawQuery = req.URL.RawQuery
			},
		}
		transportProxy.ServeHTTP(c.Writer, req)
	}
}
