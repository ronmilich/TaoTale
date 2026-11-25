package httpapi

import (
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func localAuthority(authority string) (string, bool) {
	host, port, err := net.SplitHostPort(authority)
	if err != nil {
		host, port = authority, "80"
		if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
			host = host[1 : len(host)-1]
		} else if strings.Contains(host, ":") {
			return "", false
		}
	}
	portNumber, err := strconv.ParseUint(port, 10, 16)
	if err != nil || portNumber == 0 {
		return "", false
	}
	if ip, err := netip.ParseAddr(host); err == nil && ip.IsLoopback() && ip.Zone() == "" {
		if ip.Is6() != strings.HasPrefix(authority, "[") {
			return "", false
		}
		host = ip.String()
	} else if strings.EqualFold(host, "localhost") && !strings.HasPrefix(authority, "[") {
		host = "localhost"
	} else {
		return "", false
	}
	return net.JoinHostPort(host, strconv.FormatUint(portNumber, 10)), true
}

// LocalOrigin returns the canonical authority for an HTTP loopback origin.
func LocalOrigin(origin string) (string, bool) {
	u, err := url.Parse(origin)
	if err != nil || u.Scheme != "http" || u.Host == "" || u.User != nil ||
		u.Path != "" || u.RawQuery != "" || u.ForceQuery || strings.Contains(origin, "#") {
		return "", false
	}
	return localAuthority(u.Host)
}

func localRequests(devOrigin string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authority, ok := localAuthority(c.Request.Host)
		if !ok {
			fail(c, http.StatusForbidden, "FORBIDDEN", "This request must use a loopback host.")
			return
		}
		switch c.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			c.Next()
			return
		}
		origins := c.Request.Header.Values("Origin")
		if len(origins) > 0 {
			origin, valid := LocalOrigin(origins[0])
			if len(origins) != 1 || !valid || (origin != authority && origin != devOrigin) {
				fail(c, http.StatusForbidden, "FORBIDDEN", "This request origin is not allowed.")
				return
			}
		} else if c.GetHeader("Sec-Fetch-Site") == "cross-site" {
			fail(c, http.StatusForbidden, "FORBIDDEN", "This request origin is not allowed.")
			return
		}
		c.Next()
	}
}
