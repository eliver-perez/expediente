package folderpicker

import (
	"encoding/binary"
	"testing"
)

// These are parsers for OS tables, not claims of a native selector/UAC test.
func TestRejectClientTunnelOwnerInAllListenerTables(t *testing.T) {
	table := "tcp4 0 0 127.0.0.1.18090 *.* LISTEN 131072 131072 1234 0 0x0\n"
	if !ownsDarwinListener(table, "18090", "1234") || ownsDarwinListener(table, "18090", "9999") || ownsDarwinListener(table, "18091", "1234") {
		t.Fatal("macOS listener ownership")
	}
	modern := "Proto Recv-Q Send-Q Local Address Foreign Address (state) rxbytes txbytes rhiwat shiwat process:pid state options\n" +
		"tcp4 0 0 127.0.0.1.18090 *.* LISTEN 0 0 131072 131072 AIDEX Server:1234 00100 00000008\n"
	if !ownsDarwinListener(modern, "18090", "1234") || ownsDarwinListener(modern, "18090", "9999") {
		t.Fatal("modern macOS listener ownership")
	}
	table = "0: 0100007F:46AA 00000000:0000 0A 00000000:00000000 00:00000000 00000000 400 0 12345\n"
	if !ownsLinuxListener(table, 18090, "400") || ownsLinuxListener(table, 18090, "1000") || ownsLinuxListener(table, 18091, "400") {
		t.Fatal("Linux listener ownership")
	}
	data := make([]byte, 28)
	binary.LittleEndian.PutUint32(data[:4], 1)
	binary.LittleEndian.PutUint32(data[4:8], 2)
	binary.LittleEndian.PutUint32(data[8:12], 0x0100007f)
	binary.BigEndian.PutUint16(data[12:14], 18090)
	binary.LittleEndian.PutUint32(data[24:28], 1234)
	if !ownsWindowsListener(data, 18090, 1234) || ownsWindowsListener(data, 18090, 9999) || ownsWindowsListener(data, 18091, 1234) || ownsWindowsListener(data[:15], 18090, 1234) {
		t.Fatal("Windows listener ownership")
	}
}
