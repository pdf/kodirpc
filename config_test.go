package kodirpc

import "testing"

func TestNewConfig(t *testing.T) {
	c := NewConfig()
	if c.ReadTimeout != DefaultReadTimeout {
		t.Error(`ReadTimeout field did not receive default value`)
	}
	if c.ConnectTimeout != DefaultConnectTimeout {
		t.Error(`ConnectTimeout field did not receive default value`)
	}
	if c.Reconnect != DefaultReconnect {
		t.Error(`Reconnect field did not receive default value`)
	}
	if c.ConnectBackoffScale != DefaultConnectBackoffScale {
		t.Error(`ConnectBackoffScale field did not receive default value`)
	}
}
