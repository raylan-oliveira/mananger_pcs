package main

import (
	"fmt"
	"time"
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

	// Adicionar timestamp
	result["timestamp"] = getTimestamp()

	// Coletar informações do sistema
	result["sistema"] = getSystemInfoSyscall()

	// Coletar informações de memória
	result["memoria"] = getMemoryInfoSyscall()

	// Coletar informações de disco
	result["discos"] = getDiskInfoSyscall()

	// Coletar informações de processos
	result["processos"] = getProcessInfoSyscall()

	// Coletar informações de rede
	result["rede"] = getNetworkInfoSyscall()

	return result
}

// getTimestamp retorna o timestamp atual formatado
func getTimestamp() string {
	return time.Now().Format("02/01/2006 15:04:05")
}
