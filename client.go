package kodirpc

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net"
	"sync"
	"time"
)

// NotificationHandler is a callback handler for notifications.
type NotificationHandler func(method string, data interface{})

// Error response.
type Error struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Error satisfies the error interface.
func (e *Error) Error() string {
	return fmt.Sprintf("%s [code: %d]", e.Message, e.Code)
}

type request struct {
	Version string      `json:"jsonrpc"`
	Method  *string     `json:"method,omitempty"`
	Params  interface{} `json:"params,omitempty"`
	ID      *uint64     `json:"id,omitempty"`
}

type response struct {
	Result interface{} `json:"result,omitempty"`
	Error  *Error      `json:"error,omitempty"`
	request
}

// Client is a TCP JSON-RPC client for Kodi.
type Client struct {
	address string
	config  *Config
	conn    io.ReadWriteCloser
	enc     *json.Encoder
	dec     *json.Decoder

	pending  map[uint64]chan response
	handlers map[string][]NotificationHandler
	seq      uint64

	quitChan chan struct{}

	closed bool
	sync.RWMutex
}

// Close the client connection, not further use of the Client is permitted after
// this method has been called.
func (c *Client) Close() error {
	c.Lock()
	if c.closed {
		return fmt.Errorf(`Double close`)
	}
	c.closed = true
	close(c.quitChan)
	for id := range c.pending {
		close(c.pending[id])
		delete(c.pending, id)
	}
	for method := range c.handlers {
		delete(c.handlers, method)
	}
	err := c.conn.Close()
	c.Unlock()
	return err
}

// Handle the notification method, using the specificed handler.  The handler
// will be passed the data parameter from the incoming notification.
func (c *Client) Handle(method string, handler NotificationHandler) {
	c.Lock()
	if _, ok := c.handlers[method]; !ok {
		c.handlers[method] = make([]NotificationHandler, 0, 1)
	}
	c.handlers[method] = append(c.handlers[method], handler)
	c.Unlock()
}

// Notify sends the RPC request and does not wait for a response.
func (c *Client) Notify(method string, params interface{}) error {
	_, _, err := c.call(method, params, false)
	return err
}

// Call an RPC method and return the result.
func (c *Client) Call(method string, params interface{}) (interface{}, error) {
	var res response
	id, ch, err := c.call(method, params, true)
	if err != nil {
		return nil, err
	}
	select {
	case res = <-ch:
		c.clearPending(id)
	case <-time.After(c.config.ReadTimeout):
		c.clearPending(id)
		return nil, fmt.Errorf(`Timed out`)
	case <-c.quitChan:
		return nil, fmt.Errorf(`Closing`)
	}

	if res.Error != nil {
		err = res.Error
	}
	return res.Result, err
}

func (c *Client) clearPending(id uint64) {
	c.Lock()
	close(c.pending[id])
	delete(c.pending, id)
	c.Unlock()
}

func (c *Client) call(method string, params interface{}, withResponse bool) (uint64, chan response, error) {
	var (
		ch  chan response
		id  uint64
		req = request{
			Version: `2.0`,
			Method:  &method,
			Params:  params,
		}
	)
	if withResponse {
		ch = make(chan response)
		c.Lock()
		id = c.seq
		c.seq++
		c.pending[id] = ch
		c.Unlock()
		req.ID = &id
	}
	// Block during reconnect
	c.RLock()
	err := c.enc.Encode(req)
	c.RUnlock()
	return id, ch, err
}

func (c *Client) reader() {
	var (
		res response
		err error
	)
	for {
		select {
		case <-c.quitChan:
			return
		default:
			res = response{}
			// Block during reconnect
			c.RLock()
			c.RUnlock()
			err = c.dec.Decode(&res)
			if err != nil {
				c.RLock()
				if c.closed {
					c.RUnlock()
					return
				}
				c.RUnlock()
				if _, ok := err.(net.Error); ok || err == io.EOF {
					logger.Warnf("Disconnected: %v", err)
					if c.config.Reconnect {
						if err = c.dial(); err != nil {
							if err = c.Close(); err != nil {
								logger.Errorf("Failed to clean up: %v", err)
							}
						}
					} else {
						if err = c.Close(); err != nil {
							logger.Errorf("Failed to clean up: %v", err)
						}
					}
				} else {
					logger.Warnf("Failed decoding message: %v", err)
				}
				continue
			}
			if err = c.process(res); err != nil {
				logger.Debugf(err.Error())
			}
		}
	}
}

func (c *Client) process(res response) error {
	if res.ID != nil {
		c.RLock()
		ch, ok := c.pending[*res.ID]
		if !ok {
			c.RUnlock()
			return fmt.Errorf("Unknown request ID: %d", *res.ID)
		}
		// Stay locked around the channel write so that we don't write to the
		// chan after close
		ch <- res
		c.RUnlock()
		return nil
	}
	if res.Method != nil {
		params, ok := res.Params.(map[string]interface{})
		if !ok {
			return fmt.Errorf("Received notification with malformed params: %+v", res.Params)
		}
		var handlers []NotificationHandler
		c.RLock()
		if _, ok := c.handlers[*res.Method]; !ok {
			c.RUnlock()
			return fmt.Errorf("Unclaimed notification (%s): %+v", *res.Method, res)
		}
		handlers = make([]NotificationHandler, len(c.handlers[*res.Method]))
		copy(handlers, c.handlers[*res.Method])
		c.RUnlock()

		for _, handler := range handlers {
			go handler(*res.Method, params["data"])
		}
		return nil
	}

	return fmt.Errorf("Unhandled message: %+v", res)
}

func (c *Client) dial() (err error) {
	var (
		backoff          = 10 * time.Millisecond
		attempt  float64 = 1
		duration time.Duration
	)
	c.Lock()
	defer c.Unlock()

	logger.Infof("Connecting (%s)", c.address)
	for {
		c.conn, err = net.Dial(`tcp`, c.address)
		if err != nil {
			duration = time.Duration(math.Pow(float64(backoff/time.Millisecond), attempt)) * time.Millisecond
			if duration < 0 {
				// wrapped, so just trip our timeout
				duration = c.config.ConnectTimeout + 1
			}
			if !c.config.Reconnect || (c.config.ConnectTimeout != 0 && duration > c.config.ConnectTimeout) {
				return fmt.Errorf("Could not establish connection (%s): %v", c.address, err)
			}
			logger.Debugf("Sleeping for %dms/%dms", duration/time.Millisecond, c.config.ConnectTimeout/time.Millisecond)
			time.Sleep(duration)
			attempt++
			continue
		}
		c.enc = json.NewEncoder(c.conn)
		c.dec = json.NewDecoder(c.conn)
		logger.Infof("Connected (%s)", c.address)
		return nil
	}
}

// NewClient connects to the specified address and returns the resulting Client.
func NewClient(address string, config *Config) (c *Client, err error) {
	if config == nil {
		config = NewConfig()
	}
	c = &Client{
		address:  address,
		config:   config,
		pending:  make(map[uint64]chan response),
		handlers: make(map[string][]NotificationHandler),
		quitChan: make(chan struct{}),
	}
	if err = c.dial(); err != nil {
		return nil, err
	}
	go c.reader()

	return c, nil
}
