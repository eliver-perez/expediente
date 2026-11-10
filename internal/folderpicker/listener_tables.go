package folderpicker

import (
	"encoding/binary"
	"strconv"
	"strings"
)

func ownsWindowsListener(data []byte, port int, pid uint32) bool {
	if len(data) < 4 {
		return false
	}
	count := binary.LittleEndian.Uint32(data[:4])
	if uint64(count) > uint64((len(data)-4)/24) {
		return false
	}
	found := false
	for index := uint32(0); index < count; index++ {
		row := data[4+int(index)*24 : 4+int(index+1)*24]
		address := binary.LittleEndian.Uint32(row[4:8])
		if int(binary.BigEndian.Uint16(row[8:10])) != port || (address != 0 && address != 0x0100007f) {
			continue
		}
		if binary.LittleEndian.Uint32(row[:4]) != 2 || binary.LittleEndian.Uint32(row[20:24]) != pid {
			return false
		}
		found = true
	}
	return found
}

func ownsLinuxListener(table string, port uint64, uid string) bool {
	found := false
	for _, line := range strings.Split(table, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 10 || fields[3] != "0A" {
			continue
		}
		address := strings.Split(fields[1], ":")
		if len(address) != 2 || (address[0] != "00000000" && address[0] != "0100007F") {
			continue
		}
		candidate, err := strconv.ParseUint(address[1], 16, 16)
		if err != nil || candidate != port {
			continue
		}
		if fields[7] != uid {
			return false
		}
		found = true
	}
	return found
}

func ownsDarwinListener(table, port, pid string) bool {
	found := false
	modern := strings.Contains(table, "process:pid")
	for _, line := range strings.Split(table, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 10 || !strings.HasPrefix(fields[0], "tcp") || fields[5] != "LISTEN" {
			continue
		}
		if fields[3] != "127.0.0.1."+port && fields[3] != "*."+port {
			continue
		}
		owner := fields[8]
		if modern {
			owner = ""
			for _, field := range fields[6:] {
				if colon := strings.LastIndexByte(field, ':'); colon >= 0 {
					candidate := field[colon+1:]
					if number, err := strconv.Atoi(candidate); err == nil && number > 0 {
						owner = candidate
						break
					}
				}
			}
		}
		if owner != pid {
			return false
		}
		found = true
	}
	return found
}
