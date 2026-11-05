package folderpicker

import (
	"net"
	"os"
	"strconv"
	"testing"
)

func TestNativeLinuxListenerOwnership(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	_, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	number, err := strconv.ParseUint(port, 10, 16)
	if err != nil {
		t.Fatal(err)
	}
	table, err := os.ReadFile("/proc/net/tcp")
	if err != nil {
		t.Fatal(err)
	}
	if !ownsLinuxListener(string(table), number, strconv.Itoa(os.Getuid())) || ownsLinuxListener(string(table), number, "-1") {
		t.Fatal("native listener did not match the owning account")
	}
}
