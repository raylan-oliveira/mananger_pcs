package main

import (
	"os/exec"
	"strings"
)

func getHardwareInfo() map[string]interface{} {
	info := make(map[string]interface{})

	// Obtendo informações básicas de hardware
	cmd := exec.Command("wmic", "computersystem", "get", "Manufacturer,Model,SystemFamily,SystemType", "/format:csv")
	output, err := cmd.Output()
	if err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.Contains(line, ",") && !strings.Contains(line, "Node") {
				parts := strings.Split(line, ",")
				if len(parts) >= 5 {
					info["fabricante"] = strings.TrimSpace(parts[1])
					info["modelo"] = strings.TrimSpace(parts[2])
					info["familia"] = strings.TrimSpace(parts[3])
					info["tipo_sistema"] = strings.TrimSpace(parts[4])
				}
				break
			}
		}
	}

	// Obtendo informações de BIOS
	cmd = exec.Command("wmic", "bios", "get", "Manufacturer,Name,SerialNumber,Version,SMBIOSBIOSVersion", "/format:csv")
	output, err = cmd.Output()
	if err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.Contains(line, ",") && !strings.Contains(line, "Node") {
				parts := strings.Split(line, ",")
				if len(parts) >= 6 {
					info["bios_fabricante"] = strings.TrimSpace(parts[1])
					info["bios_nome"] = strings.TrimSpace(parts[2])
					info["numero_serie"] = strings.TrimSpace(parts[3])
					info["versao_bios"] = strings.TrimSpace(parts[4])
					info["smbios_versao"] = strings.TrimSpace(parts[5])
				}
				break
			}
		}
	}

	// Garantir que os campos básicos nunca sejam nulos
	if info["fabricante"] == nil {
		// Tentar método alternativo para obter fabricante
		cmd = exec.Command("powershell", "-Command", "(Get-WmiObject -Class Win32_ComputerSystem).Manufacturer")
		output, err = cmd.Output()
		if err == nil {
			info["fabricante"] = strings.TrimSpace(string(output))
		} else {
			info["fabricante"] = "Desconhecido"
		}
	}

	if info["modelo"] == nil {
		// Tentar método alternativo para obter modelo
		cmd = exec.Command("powershell", "-Command", "(Get-WmiObject -Class Win32_ComputerSystem).Model")
		output, err = cmd.Output()
		if err == nil {
			info["modelo"] = strings.TrimSpace(string(output))
		} else {
			info["modelo"] = "Desconhecido"
		}
	}

	if info["numero_serie"] == nil {
		// Tentar método alternativo para obter número de série
		cmd = exec.Command("powershell", "-Command", "(Get-CimInstance -ClassName Win32_BIOS).SerialNumber")
		output, err = cmd.Output()
		if err == nil {
			info["numero_serie"] = strings.TrimSpace(string(output))
		} else {
			info["numero_serie"] = "Desconhecido"
		}
	}

	if info["versao_bios"] == nil {
		// Tentar método alternativo para obter versão do BIOS
		cmd = exec.Command("powershell", "-Command", "(Get-CimInstance -ClassName Win32_BIOS).Version")
		output, err = cmd.Output()
		if err == nil {
			info["versao_bios"] = strings.TrimSpace(string(output))
		} else {
			info["versao_bios"] = "Desconhecido"
		}
	}

	// Obtendo informações detalhadas do BIOS do registro do Windows
	cmd = exec.Command("reg", "query", "HKLM\\HARDWARE\\DESCRIPTION\\System\\BIOS", "/s")
	output, err = cmd.Output()
	if err == nil {
		// Criando um mapa para armazenar as informações do registro
		regInfo := make(map[string]string)
		lines := strings.Split(string(output), "\n")

		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "HKEY_") || line == "" {
				continue
			}

			parts := strings.SplitN(line, "    ", 3) // Separar por 4+ espaços
			if len(parts) >= 3 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[2])
				regInfo[key] = value

				// Usar informações do registro para preencher campos básicos se ainda estiverem vazios
				if key == "SystemManufacturer" && (info["fabricante"] == nil || info["fabricante"] == "Desconhecido") {
					info["fabricante"] = value
				}
				if key == "SystemProductName" && (info["modelo"] == nil || info["modelo"] == "Desconhecido") {
					info["modelo"] = value
				}
				if key == "SystemFamily" && info["familia"] == nil {
					info["familia"] = value
				}
				if key == "BIOSVersion" && (info["versao_bios"] == nil || info["versao_bios"] == "Desconhecido") {
					info["versao_bios"] = value
				}
				if key == "BIOSVendor" && info["bios_fabricante"] == nil {
					info["bios_fabricante"] = value
				}
			}
		}

		// Adicionando as informações do registro ao mapa de hardware
		info["bios_registro"] = regInfo
	}

	// Obtendo informações de placa-mãe
	cmd = exec.Command("wmic", "baseboard", "get", "Manufacturer,Product,Version,SerialNumber", "/format:csv")
	output, err = cmd.Output()
	if err == nil {
		lines := strings.Split(string(output), "\n")
		placaMae := make(map[string]interface{})

		for _, line := range lines {
			if strings.Contains(line, ",") && !strings.Contains(line, "Node") {
				parts := strings.Split(line, ",")
				if len(parts) >= 5 {
					placaMae["fabricante"] = strings.TrimSpace(parts[1])
					placaMae["modelo"] = strings.TrimSpace(parts[2])
					placaMae["versao"] = strings.TrimSpace(parts[3])
					placaMae["numero_serie"] = strings.TrimSpace(parts[4])
				}
				break
			}
		}

		if len(placaMae) > 0 {
			info["placa_mae"] = placaMae
		}
	}

	return info
}

func getDiskInfo() []map[string]interface{} {
	var disks []map[string]interface{}

	// Fix: Use PowerShell for more reliable disk information
	cmd := exec.Command("powershell", "-Command",
		"Get-WmiObject Win32_LogicalDisk | "+
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
