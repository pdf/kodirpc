package kodirpc

import "time"

const (
	// DefaultReadTimeout is the default time a call will wait for a response.
	DefaultReadTimeout = 5 * time.Second
	// DefaultConnectTimeout is the default time re-/connection will be
	// attempted before failure.
	DefaultConnectTimeout = 5 * time.Minute
)

// Config represents the user-configurable parameters for the client
type Config struct {
	// ReadTimeout is the time a call will wait for a response before failure.
	ReadTimeout time.Duration
	// ConnectTimeout is the time a re-/connection will be attempted before
	// failure. A value of zero attempts indefinitely.
	ConnectTimeout time.Duration
}

// NewConfig returns a config instance with default values.
func NewConfig() (c *Config) {
	return &Config{
		ReadTimeout:    DefaultReadTimeout,
		ConnectTimeout: DefaultConnectTimeout,
	}
}
