package network

import (
	"fmt"
	"net"
	"strconv"
	"unsafe"

	"golang.org/x/sys/windows"
)

var getExtendedTCPTable = windows.NewLazySystemDLL("iphlpapi.dll").NewProc("GetExtendedTcpTable")

// On Windows a closed port may take seconds to refuse a connection. The old
// 250 ms Dial probe therefore rejected free ports before bind was attempted.
// Inspect the OS listener table instead; firewall filtering and slow refusals
// cannot create false conflicts. No elevation or SO_REUSEADDR is necessary.
func checkLANPort(address string) error {
	_, rawPort, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("%w: invalid address", errPortInspection)
	}
	port, err := strconv.ParseUint(rawPort, 10, 16)
	if err != nil || port == 0 {
		return fmt.Errorf("%w: invalid port", errPortInspection)
	}
	if err = getExtendedTCPTable.Find(); err != nil {
		return fmt.Errorf("%w: IP Helper unavailable", errPortInspection)
	}
	size := uint32(16384)
	for attempt := 0; attempt < 4; attempt++ {
		if size < 4 || size > 4<<20 {
			return errPortInspection
		}
		buffer := make([]byte, size)
		// AF_INET=2; TCP_TABLE_OWNER_PID_LISTENER=3; reserved=0.
		status, _, _ := getExtendedTCPTable.Call(uintptr(unsafe.Pointer(&buffer[0])), uintptr(unsafe.Pointer(&size)), 0, 2, 3, 0)
		if status == uintptr(windows.ERROR_INSUFFICIENT_BUFFER) {
			continue
		}
		if status != 0 || size > uint32(len(buffer)) {
			return fmt.Errorf("%w: Windows status %d", errPortInspection, status)
		}
		return checkWindowsTCPTable(buffer[:size], uint16(port))
	}
	return errPortInspection
}
