package main

import (
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Variável global para armazenar as informações do systeminfo
var systemInfoCache map[string]interface{}
var systemInfoCacheTime time.Time
var systemInfoCacheMutex sync.Mutex

// getSystemInfoData executa o comando systeminfo uma vez e faz o parser do resultado
// retornando um mapa com as informações. Implementa cache para evitar múltiplas execuções.
func getSystemInfoData() map[string]interface{} {
	systemInfoCacheMutex.Lock()
	defer systemInfoCacheMutex.Unlock()

	// Verificar se o cache ainda é válido (menos de 5 minutos)
	if systemInfoCache != nil && time.Since(systemInfoCacheTime) < 5*time.Minute {
		return systemInfoCache
	}

	// Inicializar o mapa de resultados
	info := make(map[string]interface{})

	// Executar o comando systeminfo
	cmd := exec.Command("systeminfo")
	output, err := cmd.Output()
	rawSystemInfo := make(map[string]string)

	if err == nil {
		// Analisar a saída do systeminfo
		sysInfoStr := string(output)
		lines := strings.Split(sysInfoStr, "\n")

		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || !strings.Contains(line, ":") {
				continue
			}

			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				rawSystemInfo[key] = value
			}
		}

		// Extrair informações relevantes
		if hostName, ok := rawSystemInfo["Nome do host"]; ok {
			info["nome_host"] = hostName
		}
		if osName, ok := rawSystemInfo["Nome do sistema operacional"]; ok {
			info["nome_so"] = osName
		}
		info["arquitetura"] = runtime.GOARCH
		if osVersion, ok := rawSystemInfo["Versão do sistema operacional"]; ok {
			info["versao_compilacao"] = osVersion
		} else {
			// Método 1: Usando PowerShell para obter a versão do Windows
			cmd := exec.Command("powershell", "-Command", "(Get-WmiObject -class Win32_OperatingSystem).Version")
			output, err := cmd.Output()
			if err == nil && len(output) > 0 {
				info["versao_compilacao"] = strings.TrimSpace(string(output))
			} else {
				// Método 2: Usando WMIC como alternativa
				cmd = exec.Command("wmic", "os", "get", "Version", "/value")
				output, err = cmd.Output()
				if err == nil {
					version := strings.TrimSpace(string(output))
					if strings.Contains(version, "=") {
						parts := strings.Split(version, "=")
						if len(parts) >= 2 {
							info["versao_compilacao"] = strings.TrimSpace(parts[1])
						}
					}
				} else {
					// Método 3: Usando o registro do Windows
					cmd = exec.Command("powershell", "-Command",
						"Get-ItemProperty -Path 'HKLM:\\SOFTWARE\\Microsoft\\Windows NT\\CurrentVersion' | Select-Object -ExpandProperty CurrentVersion")
					output, err = cmd.Output()
					if err == nil && len(output) > 0 {
						baseVersion := strings.TrimSpace(string(output))

						// Tentar obter o número de build para uma versão mais completa
						cmd = exec.Command("powershell", "-Command",
							"Get-ItemProperty -Path 'HKLM:\\SOFTWARE\\Microsoft\\Windows NT\\CurrentVersion' | Select-Object -ExpandProperty CurrentBuildNumber")
						buildOutput, buildErr := cmd.Output()
						if buildErr == nil && len(buildOutput) > 0 {
							build := strings.TrimSpace(string(buildOutput))
							info["versao_compilacao"] = baseVersion + "." + build
						} else {
							info["versao_compilacao"] = baseVersion
						}
					} else {
						// Método 4: Usando o comando ver
						cmd = exec.Command("cmd", "/c", "ver")
						output, err = cmd.Output()
						if err == nil && len(output) > 0 {
							verOutput := strings.TrimSpace(string(output))
							// Extrair a versão usando regex
							re := regexp.MustCompile(`\[Version\s+([0-9\.]+)\]`)
							matches := re.FindStringSubmatch(verOutput)
							if len(matches) >= 2 {
								info["versao_compilacao"] = matches[1]
							} else {
								// Último recurso: usar a versão do Go
								info["versao_compilacao"] = runtime.Version()
							}
						} else {
							// Último recurso: usar a versão do Go
							info["versao_compilacao"] = runtime.Version()
						}
					}
				}
			}
		}

		if biosVersion, ok := rawSystemInfo["Versão do BIOS"]; ok {
			info["versao_bios"] = biosVersion
		}

		if domain, ok := rawSystemInfo["Domínio"]; ok {
			info["dominio"] = domain
		}

		// Adicionar informações extras que podem ser úteis
		if physicalMemory, ok := rawSystemInfo["Memória física total"]; ok {
			info["memoria_fisica_total"] = physicalMemory
		}

		if availableMemory, ok := rawSystemInfo["Memória física disponível"]; ok {
			info["memoria_fisica_disponivel"] = availableMemory
		}
	} else {
		// Retorno aos métodos existentes se o systeminfo falhar

		// Obtenha o nome do sistema operacional do registro
		cmd := exec.Command("powershell", "-Command", "Get-ItemProperty -Path 'HKLM:\\SOFTWARE\\Microsoft\\Windows NT\\CurrentVersion' -Name ProductName | Select-Object -ExpandProperty ProductName")
		output, err := cmd.Output()
		if err == nil && len(output) > 0 {
			info["nome_so"] = strings.TrimSpace(string(output))
		} else {
			// Retornar ao runtime.GOOS se a consulta do registro falhar
			info["nome_so"] = runtime.GOOS
		}

		info["arquitetura"] = runtime.GOARCH

		// Obter versão do sistema operacional
		cmd = exec.Command("powershell", "-Command", "(Get-WmiObject -class Win32_OperatingSystem).Version")
		output, err = cmd.Output()
		if err == nil {
			info["versao_compilacao"] = strings.TrimSpace(string(output))
		} else {
			// Fallback para método alternativo
			cmd = exec.Command("wmic", "os", "get", "Version", "/value")
			output, err = cmd.Output()
			if err == nil {
				version := strings.TrimSpace(string(output))
				if strings.Contains(version, "=") {
					parts := strings.Split(version, "=")
					if len(parts) >= 2 {
						info["versao_compilacao"] = strings.TrimSpace(parts[1])
					}
				}
			}
		}

		// Nota: A coleta de uptime foi movida para a seção comum acima
	}

	// Extraia o tempo de inicialização e calcule o tempo de atividade
	var uptimeInfo []string

	// Método 1: Extrair do systeminfo (já executado acima)
	if bootTimeStr, ok := rawSystemInfo["Tempo de Inicialização do Sistema"]; ok {
		// Armazenar o valor bruto sem cálculos
		info["uptime_boot_time"] = bootTimeStr
		uptimeInfo = append(uptimeInfo, fmt.Sprintf("systeminfo: %s", bootTimeStr))
	}

	// Método 2: Obter uptime em minutos diretamente do PowerShell
	cmd = exec.Command("powershell", "-Command",
		"$uptime = (Get-Date) - (Get-CimInstance -ClassName Win32_OperatingSystem).LastBootUpTime; "+
			"Write-Host $uptime.TotalMinutes")
	output, err = cmd.Output()
	if err == nil {
		uptimeMinutesStr := strings.TrimSpace(string(output))
		// Armazenar apenas no array de uptime_raw, não mais como campo separado
		uptimeInfo = append(uptimeInfo, fmt.Sprintf("powershell_minutes: %s", uptimeMinutesStr))
	}

	// Método 3: Obter uptime formatado do PowerShell
	cmd = exec.Command("powershell", "-Command",
		"$uptime = (Get-Date) - (Get-CimInstance -ClassName Win32_OperatingSystem).LastBootUpTime; "+
			"Write-Host \"$($uptime.Days)d $($uptime.Hours)h $($uptime.Minutes)m\"")
	output, err = cmd.Output()
	if err == nil {
		uptimeFormatted := strings.TrimSpace(string(output))
		// Armazenar apenas no array de uptime_raw, não mais como campo separado
		uptimeInfo = append(uptimeInfo, fmt.Sprintf("powershell_formatted: %s", uptimeFormatted))
	}

	// Armazenar todas as informações de uptime coletadas
	if len(uptimeInfo) > 0 {
		info["uptime_raw"] = uptimeInfo

		// Modificado para usar diretamente os valores do array sem criar campos separados
		// Procurar pelo valor formatado no array
		for _, item := range uptimeInfo {
			if strings.HasPrefix(item, "powershell_formatted:") {
				formattedValue := strings.TrimPrefix(item, "powershell_formatted: ")
				info["uptime"] = formattedValue
				break
			}
		}

		// Se não encontrou o valor formatado, tentar o valor em minutos
		if info["uptime"] == nil {
			for _, item := range uptimeInfo {
				if strings.HasPrefix(item, "powershell_minutes:") {
					minutesValue := strings.TrimPrefix(item, "powershell_minutes: ")
					info["uptime"] = minutesValue
					break
				}
			}
		}

		// Se ainda não encontrou, usar o boot time
		if info["uptime"] == nil && info["uptime_boot_time"] != nil {
			info["uptime"] = info["uptime_boot_time"]
		}
	}

	// Variável para armazenar o uptime em minutos
	var uptimeMinutes int64 = 0

	// Método 1: Obter uptime em minutos diretamente do PowerShell
	cmd = exec.Command("powershell", "-Command",
		"$uptime = (Get-Date) - (Get-CimInstance -ClassName Win32_OperatingSystem).LastBootUpTime; "+
			"Write-Host $uptime.TotalMinutes")
	output, err = cmd.Output()
	if err == nil {
		uptimeMinutesStr := strings.TrimSpace(string(output))
		var uptimeMinutesFloat float64
		_, err := fmt.Sscanf(uptimeMinutesStr, "%f", &uptimeMinutesFloat)
		if err == nil {
			uptimeMinutes = int64(uptimeMinutesFloat)
			// Apenas armazenamos o valor em minutos, sem formatação
			info["uptime"] = uptimeMinutes
		}
	}

	// Método 2: Fallback para uptime
	if info["uptime"] == nil {
		cmd = exec.Command("powershell", "-Command",
			"$uptime = (Get-Date) - (Get-CimInstance -ClassName Win32_OperatingSystem).LastBootUpTime; "+
				"Write-Host $uptime.TotalMinutes")
		output, err = cmd.Output()
		if err == nil {
			uptimeMinutesStr := strings.TrimSpace(string(output))
			var uptimeMinutesFloat float64
			_, parseErr := fmt.Sscanf(uptimeMinutesStr, "%f", &uptimeMinutesFloat)
			if parseErr == nil {
				uptimeMinutes = int64(uptimeMinutesFloat)
				info["uptime"] = uptimeMinutes
			}
		}
	}

	// Garantir que uptime nunca seja nulo
	if info["uptime"] == nil {
		info["uptime"] = "Desconhecido"
	}

	// Atualizar o cache
	systemInfoCache = info
	systemInfoCacheTime = time.Now()

	return info
}

// Função para coletar informações do sistema
func collectSystemInfo() (SystemInfo, error) {
	// Obter informações detalhadas do sistema
	sistemaInfo := getSystemInfoData()

	// Obter usuários logados e incluí-los no mapa sistema
	usuariosLogados := getLoggedUsers()
	if sistemaInfo == nil {
		sistemaInfo = make(map[string]interface{})
	}
	sistemaInfo["usuarios_logados"] = usuariosLogados

	// Obter a versão do agente do banco de dados
	versaoAgente, err := getCurrentVersion()
	if err != nil {
		versaoAgente = "0.0.1" // Versão padrão se não conseguir obter do banco
	}

	// Informações básicas do sistema
	info := SystemInfo{
		Sistema:   sistemaInfo,
		CPU:       getCPUInfo(),
		Memoria:   getDetailedMemoryInfo(),
		Discos:    getDiskInfo(),
		Rede:      getNetworkInfo(),
		GPU:       getGPUInfo(),
		Processos: getProcessInfo(),
		Hardware:  getHardwareInfo(),
		Agente: AgenteInfo{
			VersaoAgente:             versaoAgente,
			ServidorAtualizacao:      "",
			SystemInfoUpdateInterval: "",
			UpdateCheckInterval:      "",
		},
	}

	// Garantindo que não há valores nulos
	if info.Sistema == nil {
		info.Sistema = make(map[string]interface{})
	}
	if info.CPU == nil {
		info.CPU = make(map[string]interface{})
	}
	if info.Memoria == nil {
		info.Memoria = make(map[string]interface{})
	}
	if info.Discos == nil {
		info.Discos = []map[string]interface{}{}
	}
	if info.Rede == nil {
		info.Rede = make(map[string]interface{})
	}
	if info.Processos == nil {
		info.Processos = make(map[string]interface{})
	}

	return info, nil
}

// Funções auxiliares para coletar informações do sistema
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

func getMemoryInfo() string {
	// Usando PowerShell para obter informações de memória de forma mais confiável
	cmd := exec.Command("powershell", "-Command", "(Get-WmiObject -Class Win32_ComputerSystem).TotalPhysicalMemory")
	output, err := cmd.Output()
	if err == nil && len(output) > 0 {
		memBytes := parseUint64(strings.TrimSpace(string(output)))
		if memBytes > 0 {
			memGB := float64(memBytes) / 1024 / 1024 / 1024
			return fmt.Sprintf("%.2f GB", memGB)
		}
	}

	// Tentativa alternativa com WMIC
	cmd = exec.Command("wmic", "OS", "get", "TotalVisibleMemorySize")
	output, err = cmd.Output()
	if err == nil {
		lines := strings.Split(string(output), "\n")
		if len(lines) >= 2 {
			memKB := strings.TrimSpace(lines[1])
			if memKB != "" {
				memGB := float64(parseUint64(memKB)) / 1024 / 1024
				return fmt.Sprintf("%.2f GB", memGB)
			}
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

func getDetailedMemoryInfo() map[string]interface{} {
	info := make(map[string]interface{})

	// Comando WMIC para obter memória total e livre em formato CSV
	// wmic OS get TotalVisibleMemorySize,FreePhysicalMemory /format:csv
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

	// Se o primeiro método falhar, tente uma abordagem alternativa
	if len(info) == 0 {
		// Comando PowerShell para obter memória total e livre
		// PowerShell -Command "$os = Get-WmiObject Win32_OperatingSystem; $total = $os.TotalVisibleMemorySize; $free = $os.FreePhysicalMemory; Write-Host \"$total,$free\""
		cmd := exec.Command("powershell", "-Command",
			"$os = Get-WmiObject Win32_OperatingSystem; "+
				"$total = $os.TotalVisibleMemorySize; "+
				"$free = $os.FreePhysicalMemory; "+
				"Write-Host \"$total,$free\"")
		output, err := cmd.Output()
		if err == nil {
			parts := strings.Split(strings.TrimSpace(string(output)), ",")
			if len(parts) >= 2 {
				totalKB := parseUint64(parts[0])

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

// Função para atualizar informações dinâmicas
// updateDynamicInfo atualiza apenas as informações dinâmicas do sistema
func updateDynamicInfo(info *SystemInfo) {
	// Atualizar informações de memória
	info.Memoria = getDetailedMemoryInfo()

	// Atualizar informações de disco
	info.Discos = getDiskInfo()

	// Atualizar informações de rede
	info.Rede = getNetworkInfo()

	// Atualizar informações de processos
	info.Processos = getProcessInfo()

	// Atualizar informações dinâmicas do sistema
	sistemaInfo := getSystemInfoData()

	// Obter usuários logados e incluí-los no mapa sistema
	usuariosLogados := getLoggedUsers()
	if sistemaInfo == nil {
		sistemaInfo = make(map[string]interface{})
	}
	sistemaInfo["usuarios_logados"] = usuariosLogados

	// Atualizar o mapa sistema
	info.Sistema = sistemaInfo

	// Atualizando informações de processos
	if runtime.GOOS == "windows" {
		// Comando WMIC para obter lista de IDs de processos
		// wmic process get ProcessId
		cmd := exec.Command("wmic", "process", "get", "ProcessId")
		output, err := cmd.Output()
		if err == nil {
			lines := strings.Split(string(output), "\n")
			count := 0
			for _, line := range lines {
				if strings.TrimSpace(line) != "" && strings.TrimSpace(line) != "ProcessId" {
					count++
				}
			}
			if info.Processos == nil {
				info.Processos = make(map[string]interface{})
			}
			info.Processos["total"] = count
		}
	}
}
