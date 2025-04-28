package main

import (
	"os/exec"
	"strings"
)

func getDetailedMemoryInfo() map[string]interface{} {
	info := make(map[string]interface{})

	// Método principal: Usar PowerShell para obter memória total
	cmd := exec.Command("powershell", "-Command", "(Get-CimInstance -Class Win32_ComputerSystem).TotalPhysicalMemory")
	output, err := cmd.Output()
	if err == nil && len(output) > 0 {
		memBytes := parseUint64(strings.TrimSpace(string(output)))
		if memBytes > 0 {
			// Converter bytes para KB para manter consistência com outros métodos
			info["total"] = memBytes / 1024
		}
	}

	// Se o primeiro método falhar, tente métodos alternativos
	if _, ok := info["total"]; !ok {
		// Comando WMIC para obter memória total e livre em formato CSV
		cmd := exec.Command("wmic", "OS", "get", "TotalVisibleMemorySize,FreePhysicalMemory", "/format:csv")
		output, err := cmd.Output()
		if err == nil {
			lines := strings.Split(string(output), "\n")
			for _, line := range lines {
				if strings.Contains(line, ",") && !strings.Contains(line, "Node") {
					parts := strings.Split(line, ",")
					if len(parts) >= 3 {
						totalKB := parseUint64(strings.TrimSpace(parts[1]))
						info["total"] = totalKB
						break
					}
				}
			}
		}
	}

	// Se ainda não tiver obtido a memória total, tente outro método
	if _, ok := info["total"]; !ok {
		// Comando PowerShell alternativo
		cmd := exec.Command("powershell", "-Command",
			"$os = Get-CimInstance Win32_OperatingSystem; "+
				"$total = $os.TotalVisibleMemorySize; "+
				"Write-Host $total")
		output, err := cmd.Output()
		if err == nil {
			totalKB := parseUint64(strings.TrimSpace(string(output)))
			if totalKB > 0 {
				info["total"] = totalKB
			}
		}
	}

	// Obter informações sobre a velocidade da memória usando wmic
	// wmic memorychip get speed
	cmd = exec.Command("wmic", "memorychip", "get", "speed")
	output, err = cmd.Output()
	if err == nil {
		lines := strings.Split(string(output), "\n")
		var speeds []int
		for i, line := range lines {
			if i > 0 && strings.TrimSpace(line) != "" {
				speed := parseUint64(strings.TrimSpace(line))
				if speed > 0 {
					speeds = append(speeds, int(speed))
				}
			}
		}
		if len(speeds) > 0 {
			info["velocidades_mhz"] = speeds
			// Usar a primeira velocidade como representativa
			info["velocidade_mhz"] = speeds[0]
		}
	}

	// Obter informações mais detalhadas sobre a memória usando PowerShell
	// PowerShell -Command "Get-CimInstance -ClassName Win32_PhysicalMemory | ForEach-Object { [PSCustomObject]@{ Slot=$_.DeviceLocator; Speed=$_.Speed; ConfiguredSpeed=$_.ConfiguredClockSpeed } } | ConvertTo-Json"
	cmd = exec.Command("powershell", "-Command",
		"Get-CimInstance -ClassName Win32_PhysicalMemory | ForEach-Object { [PSCustomObject]@{ Slot=$_.DeviceLocator; Speed=$_.Speed; ConfiguredSpeed=$_.ConfiguredClockSpeed } } | ConvertTo-Json")
	output, err = cmd.Output()
	if err == nil && len(output) > 0 {
		// Armazenar a saída bruta para análise
		jsonOutput := strings.TrimSpace(string(output))

		// Processar o JSON para extrair informações estruturadas
		var memoryModules []map[string]interface{}

		// Usar PowerShell para converter o JSON para um formato mais fácil de processar
		// PowerShell -Command "$memInfo = Get-CimInstance -ClassName Win32_PhysicalMemory | Select-Object DeviceLocator, Speed, ConfiguredClockSpeed, Capacity; $memInfo | ForEach-Object { $slot = $_.DeviceLocator; $speed = $_.Speed; $configSpeed = $_.ConfiguredClockSpeed; $capacity = $_.Capacity; Write-Host \"$slot|$speed|$configSpeed|$capacity\" }"
		cmd = exec.Command("powershell", "-Command",
			"$memInfo = Get-CimInstance -ClassName Win32_PhysicalMemory | "+
				"Select-Object DeviceLocator, Speed, ConfiguredClockSpeed, Capacity; "+
				"$memInfo | ForEach-Object { "+
				"  $slot = $_.DeviceLocator; "+
				"  $speed = $_.Speed; "+
				"  $configSpeed = $_.ConfiguredClockSpeed; "+
				"  $capacity = $_.Capacity; "+
				"  Write-Host \"$slot|$speed|$configSpeed|$capacity\" "+
				"}")
		moduleOutput, err := cmd.Output()
		if err == nil && len(moduleOutput) > 0 {
			lines := strings.Split(strings.TrimSpace(string(moduleOutput)), "\n")
			for _, line := range lines {
				parts := strings.Split(strings.TrimSpace(line), "|")
				if len(parts) >= 3 {
					module := make(map[string]interface{})
					module["slot"] = strings.TrimSpace(parts[0])
					module["velocidade"] = parseUint64(strings.TrimSpace(parts[1]))
					module["velocidade_configurada"] = parseUint64(strings.TrimSpace(parts[2]))
					if len(parts) >= 4 {
						capacityBytes := parseUint64(strings.TrimSpace(parts[3]))
						module["capacidade_gb"] = float64(capacityBytes) / 1024 / 1024 / 1024
					}
					memoryModules = append(memoryModules, module)
				}
			}

			// Se temos módulos de memória, adicione-os ao resultado
			if len(memoryModules) > 0 {
				info["modulos_memoria"] = memoryModules

				// Usar a velocidade do primeiro módulo como representativa
				if len(memoryModules) > 0 && memoryModules[0]["velocidade"] != nil {
					info["velocidade_mhz"] = memoryModules[0]["velocidade"]
				}
			}
		} else {
			// Fallback: armazenar a saída JSON bruta para diagnóstico
			info["memoria_detalhes_raw"] = jsonOutput
		}
	}

	// Garantindo valores padrão se não houver dados
	if len(info) == 0 {
		info["total"] = 0.0
	}

	return info
}
