package network

import (
	"encoding/binary"
	"errors"
)

var errPortInspection = errors.New("cannot inspect TCP listeners")
var errPortOccupied = errors.New("TCP port already has a listener")

// MIB_TCPTABLE_OWNER_PID: a DWORD count followed by 24-byte IPv4 rows.
// LocalPort occupies a DWORD but its first two bytes use network byte order.
// Accepted/TIME_WAIT sockets must not be mistaken for listening sockets: the
// HTTP request that changes the mode is still active during the replacement.
func checkWindowsTCPTable(table []byte, port uint16) error {
	if len(table) < 4 {
		return errPortInspection
	}
	count := binary.LittleEndian.Uint32(table[:4])
	if uint64(count)*24+4 > uint64(len(table)) {
		return errPortInspection
	}
	for i := uint32(0); i < count; i++ {
		row := table[4+int(i)*24:]
		if binary.LittleEndian.Uint32(row[:4]) == 2 && binary.BigEndian.Uint16(row[8:10]) == port {
			return errPortOccupied
		}
	}
	return nil
}
