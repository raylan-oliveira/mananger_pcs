package main

import (
	"strings"
	"syscall"
	"unsafe"
)

type memoryStatusEx struct {
	dwLength                uint32
	dwMemoryLoad            uint32
	ullTotalPhys            uint64
	ullAvailPhys            uint64
	ullTotalPageFile        uint64
	ullAvailPageFile        uint64
	ullTotalVirtual         uint64
	ullAvailVirtual         uint64
	ullAvailExtendedVirtual uint64
}

// Estrutura para informações de memória física
type physicalMemoryArrayInfo struct {
	Use              uint16
	Location         uint16
	ErrorCorrection  uint16
	MaxCapacity      uint32
	ErrorInfoHandle  uint16
	NumMemoryDevices uint16
}

func getMemoryInfoSyscall() map[string]interface{} {
	info := make(map[string]interface{})

	// Inicializar as DLLs e procedimentos
	err := initWindowsDLLs()
	if err != nil {
		info["erro"] = err.Error()
		return info
	}

	// Verificar se temos o procedimento necessário
	if globalMemoryStatusExFn == nil {
		info["erro"] = "Função GlobalMemoryStatusEx não encontrada"
		return info
	}

	memStat := memoryStatusEx{
		dwLength: uint32(unsafe.Sizeof(memoryStatusEx{})),
	}

	ret, _, err := globalMemoryStatusExFn.Call(uintptr(unsafe.Pointer(&memStat)))
	if ret == 0 {
		info["erro"] = err.Error()
		return info
	}

	// Informações fixas sobre a memória física
	info["total"] = memStat.ullTotalPhys / 1024
	info["total_mb"] = float64(memStat.ullTotalPhys) / 1024 / 1024
	info["total_gb"] = float64(memStat.ullTotalPhys) / 1024 / 1024 / 1024

	// Informações sobre memória virtual total (fixa)
	info["virtual_total"] = memStat.ullTotalVirtual / 1024
	info["virtual_total_mb"] = float64(memStat.ullTotalVirtual) / 1024 / 1024
	info["virtual_total_gb"] = float64(memStat.ullTotalVirtual) / 1024 / 1024 / 1024

	// Informações sobre arquivo de paginação (fixa)
	info["pagefile_total"] = memStat.ullTotalPageFile / 1024
	info["pagefile_total_mb"] = float64(memStat.ullTotalPageFile) / 1024 / 1024
	info["pagefile_total_gb"] = float64(memStat.ullTotalPageFile) / 1024 / 1024 / 1024

	// Obter informações detalhadas sobre os módulos de memória
	memoryModules := getMemoryModulesInfo()
	if len(memoryModules) > 0 {
		info["modulos"] = memoryModules
	}

	// Obter informações sobre a velocidade da memória
	memorySpeed := getMemorySpeed()
	if memorySpeed > 0 {
		info["velocidade_mhz"] = memorySpeed
	}

	// Obter informações sobre o tipo de memória
	memoryType := getMemoryType()
	if memoryType != "" {
		info["tipo"] = memoryType
	}

	return info
}

// Obtém informações sobre os módulos de memória instalados
func getMemoryModulesInfo() []map[string]interface{} {
	var modules []map[string]interface{}

	// Tentar primeiro com PowerShell
	output, err := executeCommand("powershell", "-Command",
		"Get-CimInstance -ClassName Win32_PhysicalMemory | "+
			"Select-Object BankLabel, DeviceLocator, Capacity, Speed, PartNumber, Manufacturer | "+
			"ForEach-Object { "+
			"$manufacturer = if ($_.Manufacturer -eq $null -or $_.Manufacturer -eq '' -or $_.Manufacturer -like '*Unknown*') { "+
			"  (Get-CimInstance -ClassName Win32_ComputerSystem).Manufacturer "+
			"} else { $_.Manufacturer }; "+
			"$_.BankLabel + '|' + $_.DeviceLocator + '|' + $_.Capacity + '|' + $_.Speed + '|' + $_.PartNumber + '|' + $manufacturer "+
			"}")

	if err == nil && len(output) > 0 {
		lines := strings.Split(strings.TrimSpace(output), "\n")
		for _, line := range lines {
			parts := strings.Split(line, "|")
			if len(parts) >= 6 {
				module := make(map[string]interface{})

				// Informações fixas sobre o módulo
				module["banco"] = strings.TrimSpace(parts[0])
				module["slot"] = strings.TrimSpace(parts[1])

				// Capacidade em bytes (converter para GB)
				capacityBytes := parseUint64(strings.TrimSpace(parts[2]))
				module["capacidade_bytes"] = capacityBytes
				module["capacidade_gb"] = float64(capacityBytes) / 1024 / 1024 / 1024

				// Velocidade em MHz
				module["velocidade_mhz"] = parseUint64(strings.TrimSpace(parts[3]))

				// Número de peça e fabricante
				module["numero_peca"] = strings.TrimSpace(parts[4])
				
				// Fabricante específico do módulo
				manufacturer := strings.TrimSpace(parts[5])
				if manufacturer == "" || strings.Contains(strings.ToLower(manufacturer), "unknown") {
					// Tentar obter via WMIC como alternativa
					if wmicOutput, wmicErr := executeCommand("wmic", "memorychip", "get", "Manufacturer"); wmicErr == nil {
						wmicLines := strings.Split(wmicOutput, "\n")
						if len(wmicLines) > 1 {
							manufacturer = strings.TrimSpace(wmicLines[1])
						}
					}
					
					// Se ainda não conseguiu, tentar via registro
					if manufacturer == "" || strings.Contains(strings.ToLower(manufacturer), "unknown") {
						if regOpenKeyExFn != nil && regQueryValueExFn != nil && regCloseKeyFn != nil {
							const HKEY_LOCAL_MACHINE = 0x80000002
							const KEY_READ = 0x20019

							keyPath, _ := syscall.UTF16PtrFromString("HARDWARE\\DESCRIPTION\\System\\BIOS")
							var hKey syscall.Handle
							if ret, _, _ := regOpenKeyExFn.Call(
								HKEY_LOCAL_MACHINE,
								uintptr(unsafe.Pointer(keyPath)),
								0,
								KEY_READ,
								uintptr(unsafe.Pointer(&hKey)),
							); ret == 0 {
								defer regCloseKeyFn.Call(uintptr(hKey))
								if regManufacturer := getRegistryString(hKey, "SystemManufacturer"); regManufacturer != "" {
									manufacturer = regManufacturer
								}
							}
						}
					}
				}
				
				module["fabricante"] = manufacturer

				modules = append(modules, module)
			}
		}
	}

	return modules
}

// Obtém a velocidade da memória RAM
func getMemorySpeed() int {
	// Usar WMI para obter a velocidade da memória
	output, err := executeCommand("powershell", "-Command",
		"Get-CimInstance -ClassName Win32_PhysicalMemory | Select-Object -First 1 Speed")

	if err == nil && len(output) > 0 {
		// Extrair apenas o número da saída
		speed := parseUint64(strings.TrimSpace(output))
		if speed > 0 {
			return int(speed)
		}
	}

	// Método alternativo usando WMIC
	output, err = executeCommand("wmic", "memorychip", "get", "speed")
	if err == nil && len(output) > 0 {
		lines := strings.Split(output, "\n")
		if len(lines) > 1 {
			speed := parseUint64(strings.TrimSpace(lines[1]))
			if speed > 0 {
				return int(speed)
			}
		}
	}

	return 0
}

// Obtém o tipo de memória RAM
func getMemoryType() string {
	// Usar WMI para obter o tipo de memória
	output, err := executeCommand("powershell", "-Command",
		"$memType = (Get-CimInstance -ClassName Win32_PhysicalMemory | Select-Object -First 1 SMBIOSMemoryType).SMBIOSMemoryType; "+
			"switch($memType) { "+
			"26 {'DDR4'} "+
			"24 {'DDR3'} "+
			"22 {'DDR2'} "+
			"21 {'DDR'} "+
			"20 {'SDRAM'} "+
			"19 {'EDRAM'} "+
			"18 {'VRAM'} "+
			"17 {'SRAM'} "+
			"default {'Tipo ' + $memType} "+
			"}")

	if err == nil && len(output) > 0 {
		return strings.TrimSpace(output)
	}

	return ""
}

// Obtém o fabricante da memória RAM
func getMemoryManufacturer() string {
	// Usar WMI para obter o fabricante da memória
	output, err := executeCommand("powershell", "-Command",
		"Get-CimInstance -ClassName Win32_PhysicalMemory | Select-Object -First 1 -ExpandProperty Manufacturer")

	if err == nil && len(output) > 0 {
		manufacturer := strings.TrimSpace(output)

		// Verificar se a saída contém linhas indesejadas
		if strings.Contains(manufacturer, "----") || strings.Contains(manufacturer, "Unknown") {
			// Tentar método alternativo
		} else if manufacturer != "" {
			return manufacturer
		}
	}

	// Método alternativo usando WMIC
	output, err = executeCommand("wmic", "memorychip", "get", "Manufacturer")
	if err == nil && len(output) > 0 {
		lines := strings.Split(output, "\n")
		if len(lines) > 1 {
			manufacturer := strings.TrimSpace(lines[1])
			if manufacturer != "" && manufacturer != "Manufacturer" {
				return manufacturer
			}
		}
	}

	// Terceiro método: tentar obter via registro do Windows
	if regOpenKeyExFn != nil && regQueryValueExFn != nil && regCloseKeyFn != nil {
		const HKEY_LOCAL_MACHINE = 0x80000002
		const KEY_READ = 0x20019

		keyPath, _ := syscall.UTF16PtrFromString("HARDWARE\\DESCRIPTION\\System\\BIOS")

		var hKey syscall.Handle
		ret, _, _ := regOpenKeyExFn.Call(
			HKEY_LOCAL_MACHINE,
			uintptr(unsafe.Pointer(keyPath)),
			0,
			KEY_READ,
			uintptr(unsafe.Pointer(&hKey)),
		)

		if ret == 0 {
			defer regCloseKeyFn.Call(uintptr(hKey))

			// Tentar obter o fabricante da memória
			manufacturer := getRegistryString(hKey, "BIOSVendor")
			if manufacturer != "" && manufacturer != "Unknown" {
				return manufacturer
			}
		}
	}

	return "Desconhecido"
}
