package ping

import (
	"testing"
)

func TestPingv4(t *testing.T) {
	t.Skip()
	res := Ping("8.8.8.8", 3)
	if res != 0 {
		t.Fatalf("expect ping v4 to %s successed: %v", "8.8.8.8", res)
	}
}

func TestPingv6(t *testing.T) {
	t.Skip()
	res := Ping("2001:4860:4860::8888", 3)
	if res != 0 {
		t.Fatalf("expect ping v4 to %s successed: %v", "2001:4860:4860::8888", res)
	}
}
