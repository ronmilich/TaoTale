package httpapi

import (
	"bytes"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type SPA struct {
	dist  fs.FS
	index []byte
}

func NewSPA(dist fs.FS) (*SPA, error) {
	index, err := fs.ReadFile(dist, "index.html")
	if err != nil {
		return nil, err
	}
	return &SPA{dist: dist, index: index}, nil
}

func (s *SPA) serve(c *gin.Context) bool {
	r := c.Request
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return false
	}
	name := strings.TrimPrefix(r.URL.Path, "/")
	for _, segment := range strings.Split(name, "/") {
		if strings.HasPrefix(segment, ".") {
			return false
		}
	}
	switch {
	case name == "api" || strings.HasPrefix(name, "api/"),
		name == "health" || strings.HasPrefix(name, "health/"):
		return false
	case strings.HasPrefix(name, "assets/"):
		if !fs.ValidPath(name) {
			return false
		}
		info, err := fs.Stat(s.dist, name)
		if err != nil || !info.Mode().IsRegular() {
			return false
		}
		http.ServeFileFS(c.Writer, r, s.dist, name)
	case name == "assets" || (name != "index.html" && path.Ext(name) != ""):
		return false
	default:
		c.Header("Content-Type", "text/html; charset=utf-8")
		http.ServeContent(c.Writer, r, "index.html", time.Time{}, bytes.NewReader(s.index))
	}
	return true
}
