package main

import (
	"unsafe"
)

type memoryStatusEx struct {
	dwLength                uint32
	dwMemoryLoad            uint32
	ullTotalPhys            uint64
	ullAvailPhys            uint64
	ullTotalPageFile        uint64
	ullAvailPageFile        uint64
	ullTotalVirtual         uint64
	ullAvailVirtual         uint64
	ullAvailExtendedVirtual uint64
}

func getMemoryInfoSyscall() map[string]interface{} {
	info := make(map[string]interface{})

	// Inicializar as DLLs e procedimentos
	err := initWindowsDLLs()
	if err != nil {
		info["metodo"] = "falha_syscall"
		info["erro"] = err.Error()
		return info
	}

	// Verificar se temos o procedimento necessário
	if globalMemoryStatusExFn == nil {
		info["metodo"] = "falha_syscall"
		info["erro"] = "Função GlobalMemoryStatusEx não encontrada"
		return info
	}

	memStat := memoryStatusEx{
		dwLength: uint32(unsafe.Sizeof(memoryStatusEx{})),
	}

	ret, _, err := globalMemoryStatusExFn.Call(uintptr(unsafe.Pointer(&memStat)))
	if ret == 0 {
		info["metodo"] = "falha_syscall"
		info["erro"] = err.Error()
		return info
	}

	// Converter bytes para KB para manter consistência com outros métodos
	info["total"] = memStat.ullTotalPhys / 1024
	info["metodo"] = "syscall_direto_windows"

	return info
}
