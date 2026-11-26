package app

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"path/filepath"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
	"taotale/backend/internal/httpapi"
)

const DefaultListen = "127.0.0.1:8080"

type Config struct {
	Listen       string
	MediaRoot    string
	DevOrigin    string
	FrontendDist string
	Database     *pgxpool.Config
}

func LoadConfig(ctx context.Context, listen string, getenv func(string) string) (Config, error) {
	address, err := listenAddress(ctx, listen)
	if err != nil {
		return Config{}, err
	}
	mediaRoot := getenv("MEDIA_ROOT")
	if !filepath.IsAbs(mediaRoot) {
		return Config{}, errors.New("MEDIA_ROOT must be an absolute path")
	}
	frontendDist := getenv("SERVE_FRONTEND_DIST")
	if frontendDist != "" && !filepath.IsAbs(frontendDist) {
		return Config{}, errors.New("SERVE_FRONTEND_DIST must be an absolute path")
	}
	var devOrigin string
	if raw := getenv("TAOTALE_DEV_ORIGIN"); raw != "" {
		var ok bool
		if devOrigin, ok = httpapi.LocalOrigin(raw); !ok {
			return Config{}, errors.New("TAOTALE_DEV_ORIGIN must be an HTTP loopback origin")
		}
	}
	db, err := pgxpool.ParseConfig(getenv("DATABASE_URL"))
	if err != nil {
		return Config{}, errors.New("invalid PostgreSQL configuration")
	}
	return Config{Listen: address, MediaRoot: filepath.Clean(mediaRoot), DevOrigin: devOrigin, FrontendDist: frontendDist, Database: db}, nil
}

func listenAddress(ctx context.Context, listen string) (string, error) {
	host, port, err := net.SplitHostPort(listen)
	if err != nil {
		return "", errors.New("listen must include a host and numeric port")
	}
	if _, err := strconv.ParseUint(port, 10, 16); err != nil {
		return "", errors.New("listen port must be between 0 and 65535")
	}
	if host == "localhost" {
		return net.JoinHostPort("127.0.0.1", port), nil
	}
	if ip, err := netip.ParseAddr(host); err == nil && ip.IsLoopback() && ip.Zone() == "" {
		return net.JoinHostPort(ip.String(), port), nil
	}
	if host != "api" {
		return "", errors.New("listen requires a loopback address or the Compose api service name")
	}
	// Compose explicitly selects its own service address. Resolve once and pass a
	// concrete address to net.Listen rather than binding all container interfaces.
	ips, err := net.DefaultResolver.LookupNetIP(ctx, "ip4", host)
	if err != nil || len(ips) != 1 || !ips[0].IsPrivate() || ips[0].IsUnspecified() {
		return "", errors.New("listen could not resolve one private IPv4 address for the Compose api service")
	}
	return net.JoinHostPort(ips[0].String(), port), nil
}
