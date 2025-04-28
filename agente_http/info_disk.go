package main

import (
	"os/exec"
	"strings"
)

func getDiskInfo() []map[string]interface{} {
	var disks []map[string]interface{}

	// Fix: Use PowerShell for more reliable disk information
	cmd := exec.Command("powershell", "-Command",
		"Get-CimInstance Win32_LogicalDisk | "+
			"Select-Object DeviceID, FileSystem, Size, FreeSpace | "+
			"ForEach-Object { $_.DeviceID + ',' + $_.FileSystem + ',' + $_.Size + ',' + $_.FreeSpace }")
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
				deviceID := parts[0]
				fileSystem := parts[1]
				size := parseUint64(parts[2])
				freeSpace := parseUint64(parts[3])
				usedSpace := size - freeSpace

				disk := make(map[string]interface{})
				disk["dispositivo"] = deviceID
				disk["sistema_arquivos"] = fileSystem
				disk["total_gb"] = float64(size) / 1024 / 1024 / 1024
				disk["livre_gb"] = float64(freeSpace) / 1024 / 1024 / 1024
				disk["usado_gb"] = float64(usedSpace) / 1024 / 1024 / 1024
				if size > 0 {
					disk["percentual_uso"] = float64(usedSpace) * 100 / float64(size)
				} else {
					disk["percentual_uso"] = 0.0
				}

				disks = append(disks, disk)
			}
		}
	}

	// If the first method fails, try the original WMIC approach with better parsing
	if len(disks) == 0 {
		cmd := exec.Command("wmic", "logicaldisk", "get", "DeviceID,FileSystem,Size,FreeSpace", "/format:csv")
		output, err := cmd.Output()
		if err == nil {
			lines := strings.Split(string(output), "\n")
			for _, line := range lines {
				if strings.Contains(line, ",") && !strings.Contains(line, "Node") {
					parts := strings.Split(line, ",")
					if len(parts) >= 5 {
						deviceID := strings.TrimSpace(parts[1])
						fileSystem := strings.TrimSpace(parts[2])
						size := parseUint64(strings.TrimSpace(parts[3]))
						freeSpace := parseUint64(strings.TrimSpace(parts[4]))
						usedSpace := size - freeSpace

						disk := make(map[string]interface{})
						disk["dispositivo"] = deviceID
						disk["sistema_arquivos"] = fileSystem
						disk["total_gb"] = float64(size) / 1024 / 1024 / 1024
						disk["livre_gb"] = float64(freeSpace) / 1024 / 1024 / 1024
						disk["usado_gb"] = float64(usedSpace) / 1024 / 1024 / 1024
						if size > 0 {
							disk["percentual_uso"] = float64(usedSpace) * 100 / float64(size)
						} else {
							disk["percentual_uso"] = 0.0
						}

						disks = append(disks, disk)
					}
				}
			}
		}
	}

	// Garantindo que retorna pelo menos um disco vazio se nenhum for encontrado
	if len(disks) == 0 {
		disk := make(map[string]interface{})
		disk["dispositivo"] = "N/A"
		disk["sistema_arquivos"] = "N/A"
		disk["total_gb"] = 0.0
		disk["livre_gb"] = 0.0
		disk["usado_gb"] = 0.0
		disk["percentual_uso"] = 0.0
		disks = append(disks, disk)
	}

	return disks
}
