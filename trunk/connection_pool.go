package sling

import (
	"crypto/tls"
	"net/http"
)

// DefaultPoolSize is the default maximum number of outbound connections.
const DefaultPoolSize = 8

// Config contains the options for a Pool, all settings have
// sane defaults if omitted.
type Config struct {
	// PoolSize is the maximum number of connections which will
	// be opened to the server, defaults to DefaultPoolSize
	// if less then or equal to 0.
	PoolSize int

	// SkipSSLValidation should be set to true if SSL validation is
	// not desired.
	SkipSSLValidation bool
}

// ConnectionPool holds a fixed set of connections from which
// client implementations may be created.
type ConnectionPool interface {
	// HTTP returns an HTTP client with the given base URL
	// using the pool's configuration and connections.
	HTTP(url string) (HTTP, error)
}

type pool struct {
	Config
	netHTTPClient
}

// NewConnectionPool creates a new ConnectionPool using the provided
// configuration.
func NewConnectionPool(config Config) ConnectionPool {
	poolSize := config.PoolSize
	if poolSize <= 0 {
		poolSize = DefaultPoolSize
	}

	return &pool{
		Config: config,
		netHTTPClient: newThrottledHTTPClient(&http.Client{
			Transport: &http.Transport{
				MaxIdleConnsPerHost: poolSize,
				TLSClientConfig:     &tls.Config{InsecureSkipVerify: config.SkipSSLValidation},
			},
		}, poolSize),
	}
}

// NewHTTP creates a HTTP instance for the given baseURL with its own
// ConnectionPool using the provided config.
func NewHTTP(baseURL string, config Config) (HTTP, error) {
	return NewConnectionPool(config).HTTP(baseURL)
}

func (pool *pool) HTTP(baseURL string) (HTTP, error) {
	return newHTTP(baseURL, pool.netHTTPClient)
}
