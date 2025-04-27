package main

import (
	"os/exec"
	"runtime"
	"strings"
)

func getCPUModel() string {
	// Usando PowerShell para obter informações do processador de forma mais confiável
	cmd := exec.Command("powershell", "-Command", "(Get-WmiObject -Class Win32_Processor).Name")
	output, err := cmd.Output()
	if err == nil && len(output) > 0 {
		return strings.TrimSpace(string(output))
	}

	// Tentativa alternativa com WMIC
	cmd = exec.Command("wmic", "cpu", "get", "name")
	output, err = cmd.Output()
	if err == nil {
		lines := strings.Split(string(output), "\n")
		if len(lines) >= 2 {
			return strings.TrimSpace(lines[1])
		}
	}

	return "Desconhecido"
}

func getCPUInfo() map[string]interface{} {
	info := make(map[string]interface{})

	info["modelo"] = getCPUModel()
	info["nucleos"] = runtime.NumCPU()

	return info
}
