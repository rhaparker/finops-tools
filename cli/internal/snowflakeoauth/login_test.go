package snowflakeoauth

import (
	"net"
	"testing"
)

func TestListenRedirectDefaultURI(t *testing.T) {
	ln, err := listenRedirect(DefaultRedirectURI)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	addr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("addr type %T", ln.Addr())
	}
	if addr.Port != 8765 {
		t.Fatalf("port = %d, want 8765", addr.Port)
	}
}
