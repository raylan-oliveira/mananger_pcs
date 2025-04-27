package main

import (
	"time"
)

// collectAllInfo coleta todas as informações do sistema
func collectAllInfo() (SystemInfo, error) {
	var info SystemInfo

	// Coletar informações do sistema
	info.Sistema = getSystemInfo()
	info.CPU = getCPUInfo()
	info.Memoria = getDetailedMemoryInfo()
	info.Discos = getDiskInfo()
	info.GPU = getGPUInfo()
	info.Hardware = getHardwareInfo()
	info.Rede = getNetworkInfo()
	info.Processos = getProcessInfo()
	info.Agente = getAgentInfo()

	// Atualizar o cache
	cachedSystemInfo = info
	lastUpdateTime = time.Now()

	return info, nil
}
