package main

import (
	"fmt"
)

// getAllSyscallInfo coleta todas as informações do sistema usando syscalls diretos
func getAllSyscallInfo() map[string]interface{} {
	// Inicializar as DLLs do Windows
	err := initWindowsDLLs()
	if err != nil {
		fmt.Printf("Erro ao inicializar DLLs: %v\n", err)
		return map[string]interface{}{
			"erro": fmt.Sprintf("Falha ao inicializar DLLs: %v", err),
		}
	}

	// Criar o mapa de resultado
	result := make(map[string]interface{})

	// Coletar informações do sistema
	result["sistema"] = getSystemInfoSyscall()

	// Coletar informações do CPU
	result["cpu"] = getCPUInfoSyscall()

	// Coletar informações de Hardware
	result["hardware"] = getHardwareInfoSyscall()

	// Coletar informações de memória
	result["memoria"] = getMemoryInfoSyscall()

	// Coletar informações da GPU
	result["gpu"] = getGPUInfoSyscall()

	// Coletar informações de disco
	result["disco"] = getDiskInfoSyscall()

	// Coletar informações de rede
	result["rede"] = getNetworkInfoSyscall()

	return result
}
