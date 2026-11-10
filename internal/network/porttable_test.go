package network

import (
	"encoding/binary"
	"errors"
	"testing"
)

func TestWindowsListenerTable(t *testing.T) {
	table := make([]byte, 4+24*3)
	binary.LittleEndian.PutUint32(table, 3)
	for i, state := range []uint32{5, 11, 2} { // ESTABLISHED, TIME_WAIT, LISTEN
		row := table[4+i*24:]
		binary.LittleEndian.PutUint32(row, state)
		binary.BigEndian.PutUint16(row[8:10], 18090)
	}
	if !errors.Is(checkWindowsTCPTable(table, 18090), errPortOccupied) {
		t.Fatal("listener conflict missed")
	}
	if err := checkWindowsTCPTable(table, 18091); err != nil {
		t.Fatal("unrelated port rejected", err)
	}
	binary.LittleEndian.PutUint32(table, 2)
	if err := checkWindowsTCPTable(table, 18090); err != nil {
		t.Fatal("accepted request mistaken for listener", err)
	}
	for _, invalid := range [][]byte{nil, {1, 0, 0, 0}, {255, 255, 255, 255}} {
		if !errors.Is(checkWindowsTCPTable(invalid, 18090), errPortInspection) {
			t.Fatal("invalid OS table accepted")
		}
	}
	if err := checkWindowsTCPTable([]byte{0, 0, 0, 0}, 18090); err != nil {
		t.Fatal("empty listener table rejected", err)
	}
}
