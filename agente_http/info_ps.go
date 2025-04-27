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
		"Get-Process | Sort-Object CPU -Descending | Select-Object -First 5 | "+
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
		"Get-Process | Sort-Object WorkingSet64 -Descending | Select-Object -First 5 | "+
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
