package main

import (
	"os/exec"
	"strings"
)

func getProcessInfo() map[string]interface{} {
	info := make(map[string]interface{})
	
	// Removed total process count section
	
	// Top 5 processos por CPU
	var topCPU []map[string]interface{}
	cmd := exec.Command("powershell", "-Command", 
		"Get-Process | Sort-Object CPU -Descending | Select-Object -First 5 | " +
		"ForEach-Object { $_.ProcessName + ',' + $_.Id + ',' + $(if($_.CPU -eq $null){'0'}else{$_.CPU}) + ',' + $_.WorkingSet64 }")
	output, err := cmd.Output()
	if err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			
			parts := strings.Split(line, ",")
			if len(parts) >= 4 {
				proc := make(map[string]interface{})
				proc["nome"] = parts[0]
				proc["pid"] = parts[1]
				proc["cpu"] = parts[2]
				
				// Convert memory to MB
				memBytes := parseUint64(parts[3])
				proc["memoria_mb"] = float64(memBytes) / 1024 / 1024
				
				topCPU = append(topCPU, proc)
			}
		}
	}
	info["top_5_cpu"] = topCPU
	
	// Top 5 processos por memória
	var topMem []map[string]interface{}
	cmd = exec.Command("powershell", "-Command", 
		"Get-Process | Sort-Object WorkingSet64 -Descending | Select-Object -First 5 | " +
		"ForEach-Object { $_.ProcessName + ',' + $_.Id + ',' + $(if($_.CPU -eq $null){'0'}else{$_.CPU}) + ',' + $_.WorkingSet64 }")
	output, err = cmd.Output()
	if err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			
			parts := strings.Split(line, ",")
			if len(parts) >= 4 {
				proc := make(map[string]interface{})
				proc["nome"] = parts[0]
				proc["pid"] = parts[1]
				proc["cpu"] = parts[2]
				
				// Convert memory to MB
				memBytes := parseUint64(parts[3])
				proc["memoria_mb"] = float64(memBytes) / 1024 / 1024
				
				topMem = append(topMem, proc)
			}
		}
	}
	info["top_5_memoria"] = topMem
	
	return info
}

// Removed duplicate parseUint64 function

func getLoggedUsers() []string {
	var users []string
	
	// Obter usuários logados usando PowerShell
	cmd := exec.Command("powershell", "-Command", "Get-WmiObject -Class Win32_ComputerSystem | Select-Object -ExpandProperty UserName")
	output, err := cmd.Output()
	if err == nil {
		username := strings.TrimSpace(string(output))
		if username != "" {
			users = append(users, username)
		}
	}
	
	// Método alternativo para obter mais usuários logados
	cmd = exec.Command("powershell", "-Command", "query user")
	output, err = cmd.Output()
	if err == nil {
		lines := strings.Split(string(output), "\n")
		for i, line := range lines {
			if i == 0 { // Pular cabeçalho
				continue
			}
			
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			
			fields := strings.Fields(line)
			if len(fields) > 0 {
				username := fields[0]
				if username != "" && !contains(users, username) {
					users = append(users, username)
				}
			}
		}
	}
	
	return users
}

// Função auxiliar para verificar se um slice contém um valor
func contains(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}