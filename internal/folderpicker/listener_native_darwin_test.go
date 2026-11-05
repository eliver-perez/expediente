package folderpicker

import (
	"net"
	"os"
	"os/exec"
	"strconv"
	"testing"
)

func TestNativeDarwinListenerOwnership(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	_, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	table, err := exec.Command("/usr/sbin/netstat", "-anv", "-p", "tcp").Output()
	if err != nil {
		t.Fatal(err)
	}
	if !ownsDarwinListener(string(table), port, strconv.Itoa(os.Getpid())) || ownsDarwinListener(string(table), port, "-1") {
		t.Fatal("native listener did not match the owning process")
	}
}
