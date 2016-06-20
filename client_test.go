package kodirpc

import (
	"fmt"
	"io"
	"math/rand"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TODO: Test reconnect, timeouts, close

func TestNewClient(t *testing.T) {
	address := testListener(t, nil)
	_, err := NewClient(address, NewConfig())
	assert.Nil(t, err)
}

func TestNotify(t *testing.T) {
	assert := assert.New(t)
	request := []byte(`{"jsonrpc":"2.0","method":"Test.Notify","params":{"hello":"world"}}`)
	done := make(chan struct{})
	address := testListener(t, func(rw io.ReadWriter) {
		b := make([]byte, len(request))
		n, err := rw.Read(b)
		assert.Nil(err)
		assert.Equal(n, len(request), `Encoded request length mismatch`)
		assert.Equal(b, request, `Encoded request does not match`)
		close(done)
	})

	client, err := NewClient(address, NewConfig())
	assert.Nil(err)

	err = client.Notify(`Test.Notify`, map[string]string{`hello`: `world`})
	assert.Nil(err)
	select {
	case <-time.After(DefaultReadTimeout):
		t.Error(`Timed out sending data`)
	case <-done:
		return
	}
}

func TestCall(t *testing.T) {
	assert := assert.New(t)
	callTests := []struct {
		method   string
		params   interface{}
		request  []byte
		response []byte
		result   interface{}
		err      error
	}{
		{
			`Test.Call.1`,
			map[string]string{`hello`: `world`},
			[]byte(`{"jsonrpc":"2.0","method":"Test.Call.1","params":{"hello":"world"},"id":0}`),
			[]byte(`{"result":"hello","id":0}`),
			`hello`,
			nil,
		},
		{
			`Test.Call.2`,
			map[string]string{`hello`: `world`},
			[]byte(`{"jsonrpc":"2.0","method":"Test.Call.2","params":{"hello":"world"},"id":0}`),
			[]byte(`{"result":{"hello":"world"},"id":0}`),
			map[string]interface{}{`hello`: `world`},
			nil,
		},
		{
			`Test.Call.3`,
			map[string]string{`hello`: `world`},
			[]byte(`{"jsonrpc":"2.0","method":"Test.Call.3","params":{"hello":"world"},"id":0}`),
			[]byte(`{"result":{"hello":"error"},"error":{"code":404,"message":"Not found"},"id":0}`),
			map[string]interface{}{`hello`: `error`},
			&Error{Code: 404, Message: `Not found`},
		},
	}

	for _, ct := range callTests {
		address := testListener(t, func(rw io.ReadWriter) {
			b := make([]byte, len(ct.request))
			n, err := rw.Read(b)
			assert.Nil(err)
			assert.Equal(n, len(ct.request), `Encoded request length mismatch`)
			assert.Equal(b, ct.request, `Encoded request does not match`)
			_, err = rw.Write(ct.response)
			assert.Nil(err)
		})

		client, err := NewClient(address, NewConfig())
		assert.Nil(err)

		result, err := client.Call(ct.method, ct.params)
		if ct.err == nil {
			assert.Nil(err)
		} else {
			assert.EqualError(err, ct.err.Error())
		}
		assert.Equal(result, ct.result)
	}
}

func TestHandler(t *testing.T) {
	assert := assert.New(t)
	notification := []byte(`{"jsonrpc":"2.0","method":"Test.Handler","params":{"data":"hello"}}`)
	method := `Test.Handler`
	data := `hello`
	run := make(chan struct{})
	done := make(chan struct{})
	address := testListener(t, func(rw io.ReadWriter) {
		<-run
		_, err := rw.Write(notification)
		assert.Nil(err)
	})

	client, err := NewClient(address, NewConfig())
	assert.Nil(err)
	client.Handle(`Test.Handler`, func(m string, d interface{}) {
		assert.Equal(m, method)
		assert.Equal(d, data)
		close(done)
	})
	close(run)
	select {
	case <-time.After(DefaultReadTimeout):
		t.Error(`Timed out waiting for notification`)
	case <-done:
	}
}

// testListener sets up a TCP socket on a random high port and returns the
// address. If a handler is provided, it will be passed the connection once a
// client connects
func testListener(t *testing.T, handler func(io.ReadWriter)) string {
	assert := assert.New(t)
	require := require.New(t)
	address := fmt.Sprintf("127.0.0.1:%d", rand.Int31n(16384)+20000)
	l, err := net.Listen(`tcp4`, address)
	require.Nil(err)

	go func() {
		c, err := l.Accept()
		require.Nil(err)
		defer func() {
			assert.Nil(c.Close())
		}()

		if handler != nil {
			handler(c)
		}
	}()

	return address
}
