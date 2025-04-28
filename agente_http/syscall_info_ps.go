package main

import (
	"syscall"
	"unsafe"
)

// Estruturas para informações de processos via syscall
type processEntry32 struct {
	dwSize              uint32
	cntUsage            uint32
	th32ProcessID       uint32
	th32DefaultHeapID   uintptr
	th32ModuleID        uint32
	cntThreads          uint32
	th32ParentProcessID uint32
	pcPriClassBase      int32
	dwFlags             uint32
	szExeFile           [syscall.MAX_PATH]uint16
}

type processMemoryCounters struct {
	cb                         uint32
	PageFaultCount             uint32
	PeakWorkingSetSize         uintptr
	WorkingSetSize             uintptr
	QuotaPeakPagedPoolUsage    uintptr
	QuotaPagedPoolUsage        uintptr
	QuotaPeakNonPagedPoolUsage uintptr
	QuotaNonPagedPoolUsage     uintptr
	PagefileUsage              uintptr
	PeakPagefileUsage          uintptr
}

func getProcessInfoSyscall() map[string]interface{} {
	info := make(map[string]interface{})

	// Inicializar as DLLs e procedimentos
	err := initWindowsDLLs()
	if err != nil {
		info["erro"] = err.Error()
		return info
	}

	// Verificar se temos os procedimentos necessários
	if createToolhelp32SnapshotFn == nil || process32FirstFn == nil || 
	   process32NextFn == nil || getProcessMemoryInfoFn == nil || 
	   openProcessFn == nil || closeHandleFn == nil {
		info["erro"] = "Não foi possível carregar funções de processos"
		return info
	}

	// Criar um snapshot dos processos
	hSnapshot, _, _ := createToolhelp32SnapshotFn.Call(0x2, 0) // TH32CS_SNAPPROCESS = 0x2
	if hSnapshot == uintptr(syscall.InvalidHandle) {
		return info
	}
	defer closeHandleFn.Call(hSnapshot)

	// Estrutura para armazenar informações do processo
	var pe processEntry32
	pe.dwSize = uint32(unsafe.Sizeof(pe))

	// Obter o primeiro processo
	ret, _, _ := process32FirstFn.Call(hSnapshot, uintptr(unsafe.Pointer(&pe)))
	if ret == 0 {
		return info
	}

	// Armazenar informações de todos os processos
	var processes []map[string]interface{}

	// Constantes para OpenProcess
	const PROCESS_QUERY_INFORMATION = 0x0400
	const PROCESS_VM_READ = 0x0010

	for {
		proc := make(map[string]interface{})
		proc["nome"] = syscall.UTF16ToString(pe.szExeFile[:])
		proc["pid"] = int(pe.th32ProcessID)
		proc["threads"] = int(pe.cntThreads)

		// Tentar obter informações de memória
		hProcess, _, _ := openProcessFn.Call(PROCESS_QUERY_INFORMATION|PROCESS_VM_READ, 0, uintptr(pe.th32ProcessID))
		if hProcess != 0 && hProcess != uintptr(syscall.InvalidHandle) {
			var pmc processMemoryCounters
			pmc.cb = uint32(unsafe.Sizeof(pmc))

			ret, _, _ := getProcessMemoryInfoFn.Call(hProcess, uintptr(unsafe.Pointer(&pmc)), uintptr(unsafe.Sizeof(pmc)))
			if ret != 0 {
				proc["memoria_mb"] = float64(pmc.WorkingSetSize) / 1024 / 1024
			}

			closeHandleFn.Call(hProcess)
		}

		processes = append(processes, proc)

		// Obter o próximo processo
		ret, _, _ := process32NextFn.Call(hSnapshot, uintptr(unsafe.Pointer(&pe)))
		if ret == 0 {
			break
		}
	}

	// Ordenar processos por uso de memória (simplificado - na prática seria necessário implementar a ordenação)
	// Aqui apenas pegamos os primeiros 5 processos como exemplo
	var topMem []map[string]interface{}
	count := 0
	for _, proc := range processes {
		if count >= 5 {
			break
		}
		if _, ok := proc["memoria_mb"]; ok {
			topMem = append(topMem, proc)
			count++
		}
	}

	info["top_5_memoria"] = topMem
	info["metodo"] = "syscall_direto_windows"

	return info
}
