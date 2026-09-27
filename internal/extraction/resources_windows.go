package extraction

import (
	"encoding/binary"
	"golang.org/x/sys/windows"
	"math/bits"
	"unsafe"
)

func detectPlatformResources(r *Resources) {
	kernel := windows.NewLazySystemDLL("kernel32.dll")
	memory := struct {
		Length, Load                                                                         uint32
		Total, Available, TotalPage, AvailablePage, TotalVirtual, AvailableVirtual, Extended uint64
	}{}
	memory.Length = uint32(unsafe.Sizeof(memory))
	if ok, _, _ := kernel.NewProc("GlobalMemoryStatusEx").Call(uintptr(unsafe.Pointer(&memory))); ok != 0 {
		r.MemoryBytes = memory.Total
		r.AvailableMemoryBytes = memory.Available
	}
	processorInfo := kernel.NewProc("GetLogicalProcessorInformationEx")
	var length uint32
	processorInfo.Call(0, 0, uintptr(unsafe.Pointer(&length)))
	if length > 0 && length < 1<<20 {
		buffer := make([]byte, length)
		if ok, _, _ := processorInfo.Call(0, uintptr(unsafe.Pointer(&buffer[0])), uintptr(unsafe.Pointer(&length))); ok != 0 {
			for offset := 0; offset+8 <= int(length); {
				size := int(binary.LittleEndian.Uint32(buffer[offset+4:]))
				if size < 8 || offset+size > int(length) {
					break
				}
				if binary.LittleEndian.Uint32(buffer[offset:]) == 0 {
					r.Physical++
				}
				offset += size
			}
		}
	}
	var processMask, systemMask uintptr
	if ok, _, _ := kernel.NewProc("GetProcessAffinityMask").Call(uintptr(windows.CurrentProcess()), uintptr(unsafe.Pointer(&processMask)), uintptr(unsafe.Pointer(&systemMask))); ok != 0 && processMask != 0 {
		r.AvailableCPUs = min(r.AvailableCPUs, bits.OnesCount64(uint64(processMask)))
	}
	var job windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION
	if windows.QueryInformationJobObject(0, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&job)), uint32(unsafe.Sizeof(job)), nil) == nil {
		for _, limit := range []struct {
			flag  uint32
			value uintptr
		}{{windows.JOB_OBJECT_LIMIT_PROCESS_MEMORY, job.ProcessMemoryLimit}, {windows.JOB_OBJECT_LIMIT_JOB_MEMORY, job.JobMemoryLimit}} {
			if job.BasicLimitInformation.LimitFlags&limit.flag != 0 && limit.value > 0 {
				r.MemoryBytes = min(r.MemoryBytes, uint64(limit.value))
				r.AvailableMemoryBytes = min(r.AvailableMemoryBytes, uint64(limit.value)/2)
			}
		}
		r.Notes = append(r.Notes, "Se detectó un Job Object; se consideran sus límites de memoria accesibles.")
	}
	var cpu struct{ Flags, Rate uint32 }
	if windows.QueryInformationJobObject(0, windows.JobObjectCpuRateControlInformation, uintptr(unsafe.Pointer(&cpu)), uint32(unsafe.Sizeof(cpu)), nil) == nil && cpu.Flags&1 != 0 && cpu.Flags&4 != 0 && cpu.Rate > 0 {
		r.AvailableCPUs = min(r.AvailableCPUs, max(1, r.Logical*int(cpu.Rate)/10000))
	}
	r.Notes = append(r.Notes, "Se consideran afinidad y cuotas accesibles; Job Objects anidados o políticas externas pueden imponer restricciones adicionales.")
}
