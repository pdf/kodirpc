[![Build Status](https://travis-ci.org/pdf/kodirpc.svg?branch=master)](https://travis-ci.org/pdf/kodirpc) [![GoDoc](https://godoc.org/github.com/pdf/kodirpc?status.svg)](http://godoc.org/github.com/pdf/kodirpc) ![License-MIT](http://img.shields.io/badge/license-MIT-red.svg)

# kodirpc
--
    import "github.com/pdf/kodirpc"

Package kodirpc is a JSON-RPC client for the TCP socket of the Kodi home theatre
software.

## Usage

```go
const (
	// DefaultReadTimeout is the default time a call will wait for a response.
	DefaultReadTimeout = 5 * time.Second
	// DefaultConnectTimeout is the default time re-/connection will be
	// attempted before failure.
	DefaultConnectTimeout = 5 * time.Minute
	// DefaultReconnect determines whether the client reconnects by default.
	DefaultReconnect = true
	// DefaultConnectBackoffScale is the default back-off scaling factor
	DefaultConnectBackoffScale = 2
)
```

#### func  SetLogger

```go
func SetLogger(l LevelledLogger)
```
SetLogger enables logging for the library and wraps the supplied logger with a
logPrefixer to denote locally generated logs

#### type Client

```go
type Client struct {
	sync.RWMutex
}
```

Client is a TCP JSON-RPC client for Kodi.

#### func  NewClient

```go
func NewClient(address string, config *Config) (c *Client, err error)
```
NewClient connects to the specified address and returns the resulting Client.

#### func (*Client) Call

```go
func (c *Client) Call(method string, params interface{}) (interface{}, error)
```
Call an RPC method and return the result.

#### func (*Client) Close

```go
func (c *Client) Close() error
```
Close the client connection, not further use of the Client is permitted after
this method has been called.

#### func (*Client) Handle

```go
func (c *Client) Handle(method string, handler NotificationHandler)
```
Handle the notification method, using the specificed handler. The handler will
be passed the data parameter from the incoming notification.

#### func (*Client) Notify

```go
func (c *Client) Notify(method string, params interface{}) error
```
Notify sends the RPC request and does not wait for a response.

#### type Config

```go
type Config struct {
	// ReadTimeout is the time a call will wait for a response before failure.
	ReadTimeout time.Duration
	// ConnectTimeout is the time a re-/connection will be attempted before
	// failure. A value of zero attempts indefinitely.
	ConnectTimeout time.Duration
	// Reconnect determines whether the client will attempt to reconnect on
	// connection failure
	Reconnect bool
	// ConnectBackoffScale sets the scaling factor for back-off on failed
	// connection attempts
	ConnectBackoffScale int
}
```

Config represents the user-configurable parameters for the client

#### func  NewConfig

```go
func NewConfig() (c *Config)
```
NewConfig returns a config instance with default values.

#### type Error

```go
type Error struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}
```

Error response.

#### func (*Error) Error

```go
func (e *Error) Error() string
```
Error satisfies the error interface.

#### type LevelledLogger

```go
type LevelledLogger interface {
	// Debugf handles debug level messages
	Debugf(format string, args ...interface{})
	// Infof handles info level messages
	Infof(format string, args ...interface{})
	// Warnf handles warn level messages
	Warnf(format string, args ...interface{})
	// Errorf handles error level messages
	Errorf(format string, args ...interface{})
	// Fatalf handles fatal level messages, and must exit the application
	Fatalf(format string, args ...interface{})
	// Panicf handles debug level messages, and must panic the application
	Panicf(format string, args ...interface{})
}
```

LevelledLogger represents a minimal levelled logger

#### type NotificationHandler

```go
type NotificationHandler func(method string, data interface{})
```

NotificationHandler is a callback handler for notifications.
