package main

import (
	"time"
)

// collectAllInfoSyscall coleta todas as informações do sistema
func collectAllInfoSyscall() (SystemInfo, error) {
	var info SystemInfo

	// Coletar informações do sistema
	info.Sistema = getSystemInfoSyscall()
	info.CPU = getCPUInfoSyscall()
	info.Memoria = getMemoryInfoSyscall()
	info.Discos = getDiskInfoSyscall()
	info.GPU = getGPUInfoSyscall()
	info.Hardware = getHardwareInfoSyscall()
	info.Rede = getNetworkInfoSyscall()
	info.Agente = getAgentInfo()

	// Atualizar o cache
	cachedSystemInfo = info
	lastUpdateTime = time.Now()

	return info, nil
}
