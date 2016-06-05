package kodirpc

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

// NotificationFunc is a callback handler for notifications
type NotificationFunc func(data interface{})

// Error response
type Error struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Error satisfies the error interface
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

// Client is a TCP JSON-RPC client for Kodi
type Client struct {
	conn io.ReadWriteCloser
	enc  *json.Encoder
	dec  *json.Decoder

	pending  map[uint64]chan response
	handlers map[string][]NotificationFunc
	seq      uint64

	timeout time.Duration

	quitChan chan struct{}

	sync.RWMutex
}

// Close the client connection, not further use of the Client is permitted after
// this method has been called
func (c *Client) Close() error {
	close(c.quitChan)
	return c.conn.Close()
}

// Register a notification handler for the specified method
func (c *Client) Register(method string, fun NotificationFunc) {
	c.Lock()
	if _, ok := c.handlers[method]; !ok {
		c.handlers[method] = make([]NotificationFunc, 0, 1)
	}
	c.handlers[method] = append(c.handlers[method], fun)
	c.Unlock()
}

// Notify sends the RPC request and does not wait for a response
func (c *Client) Notify(method string, params interface{}) error {
	_, _, err := c.call(method, params, false)
	return err
}

// Call an RPC method and return the result
func (c *Client) Call(method string, params interface{}) (interface{}, error) {
	var res response
	id, ch, err := c.call(method, params, true)
	if err != nil {
		return nil, err
	}
	timeout := time.After(c.timeout)
	select {
	case res = <-ch:
		c.clearPending(id)
	case <-timeout:
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
		ch chan response
		id uint64
	)
	req := request{
		Version: `2.0`,
		Method:  &method,
		Params:  params,
	}
	if withResponse {
		ch = make(chan response)
		c.Lock()
		id = c.seq
		c.seq = c.seq + 1
		c.pending[id] = ch
		c.Unlock()
		req.ID = &id
	}
	return id, ch, c.enc.Encode(req)
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
			if err = c.dec.Decode(&res); err != nil {
				logger.Warnf("Failed decoding message: %+v", err)
			} else {
				if err = c.process(res); err != nil {
					logger.Debugf("Failed processing message: %+v", err)
				}
			}
		}
	}
}

func (c *Client) process(res response) error {
	if res.ID != nil {
		c.RLock()
		ch, ok := c.pending[*res.ID]
		c.RUnlock()
		if !ok {
			return fmt.Errorf("Unknown request ID: %d", *res.ID)
		}
		ch <- res
		return nil
	}
	if res.Method != nil {
		params, ok := res.Params.(map[string]interface{})
		if !ok {
			return fmt.Errorf("Received notification with malformed params: %+v", res.Params)
		}
		c.RLock()
		for _, fun := range c.handlers[*res.Method] {
			fun(params["data"])
		}
		c.RUnlock()
		return nil
	}

	return fmt.Errorf("Unhandled message: %+v", res)
}

// NewClient connects to the specified address and returns the resulting Client
func NewClient(address string, timeout time.Duration) (c *Client, err error) {
	c = &Client{
		pending:  make(map[uint64]chan response),
		handlers: make(map[string][]NotificationFunc),
		quitChan: make(chan struct{}),
		timeout:  timeout,
	}
	c.conn, err = net.Dial(`tcp`, address)
	if err != nil {
		return nil, fmt.Errorf("Could not establish connection (%s): %v", address, err)
	}
	c.enc = json.NewEncoder(c.conn)
	c.dec = json.NewDecoder(c.conn)
	go c.reader()

	return c, nil
}
