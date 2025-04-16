package main

import (
	"os/exec"
	"strings"
)

func getGPUInfo() interface{} {
	var gpus []map[string]interface{}
	
	// Tentativa 1: Usando PowerShell para obter informações da GPU
	cmd := exec.Command("powershell", "-Command", 
		"Get-WmiObject Win32_VideoController | " +
		"ForEach-Object { $_.Name + ',' + $_.AdapterRAM + ',' + $_.DriverVersion }")
	output, err := cmd.Output()
	if err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			
			parts := strings.Split(line, ",")
			if len(parts) >= 3 {
				gpu := make(map[string]interface{})
				gpu["nome"] = parts[0]
				
				// Converter RAM para GB
				ramBytes := parseUint64(parts[1])
				gpu["memoria_gb"] = float64(ramBytes) / 1024 / 1024 / 1024
				
				gpu["versao_driver"] = parts[2]
				
				gpus = append(gpus, gpu)
			}
		}
	}
	
	// Tentativa 2: Usando WMIC se o PowerShell falhar
	if len(gpus) == 0 {
		cmd := exec.Command("wmic", "path", "win32_VideoController", "get", "Name,AdapterRAM,DriverVersion", "/format:csv")
		output, err := cmd.Output()
		if err == nil {
			lines := strings.Split(string(output), "\n")
			for _, line := range lines {
				if strings.Contains(line, ",") && !strings.Contains(line, "Node") {
					parts := strings.Split(line, ",")
					if len(parts) >= 4 {
						gpu := make(map[string]interface{})
						gpu["nome"] = strings.TrimSpace(parts[1])
						
						// Converter RAM para GB
						ramBytes := parseUint64(strings.TrimSpace(parts[2]))
						gpu["memoria_gb"] = float64(ramBytes) / 1024 / 1024 / 1024
						
						gpu["versao_driver"] = strings.TrimSpace(parts[3])
						
						gpus = append(gpus, gpu)
					}
				}
			}
		}
	}
	
	// Se não conseguiu obter informações, retornar mensagem de erro
	if len(gpus) == 0 {
		return "Não foi possível obter informações da GPU"
	}
	
	return gpus
}