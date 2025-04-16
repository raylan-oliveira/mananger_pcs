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
		info["ray"] = "deu certo"
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

		// Extraia o tempo de inicialização e calcule o tempo de atividade
		if bootTimeStr, ok := rawSystemInfo["Tempo de Inicialização do Sistema"]; ok {
			// Parse boot time in format "14/04/2025, 10:17:03"
			bootTime, err := time.Parse("02/01/2006, 15:04:05", bootTimeStr)
			if err == nil {
				uptime := time.Since(bootTime)
				info["uptime"] = int64(uptime.Minutes())
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
	}

	// Garantir que uptime nunca seja nulo
	if info["uptime"] == nil {
		info["uptime"] = 0
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

	// Obter a versão do agente do banco de dados
	versaoAgente, err := getCurrentVersion()
	if err != nil {
		versaoAgente = "1.0.0" // Versão padrão se não conseguir obter do banco
	}

	// Informações básicas do sistema
	info := SystemInfo{
		Sistema:         sistemaInfo,
		CPU:             getCPUInfo(),
		Memoria:         getDetailedMemoryInfo(),
		Discos:          getDiskInfo(),
		Rede:            getNetworkInfo(),
		GPU:             getGPUInfo(),
		Processos:       getProcessInfo(),
		UsuariosLogados: getLoggedUsers(),
		Hardware:        getHardwareInfo(),
		VersaoAgente:    versaoAgente,
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
	if info.UsuariosLogados == nil {
		info.UsuariosLogados = []string{}
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

	// Fix: Adjust the WMIC command to ensure proper output parsing
	cmd := exec.Command("wmic", "OS", "get", "TotalVisibleMemorySize,FreePhysicalMemory", "/format:csv")
	output, err := cmd.Output()
	if err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.Contains(line, ",") && !strings.Contains(line, "Node") {
				parts := strings.Split(line, ",")
				if len(parts) >= 3 {
					totalKB := parseUint64(strings.TrimSpace(parts[1]))
					freeKB := parseUint64(strings.TrimSpace(parts[2]))
					usedKB := totalKB - freeKB

					info["total_gb"] = float64(totalKB) / 1024 / 1024
					info["disponivel_gb"] = float64(freeKB) / 1024 / 1024
					info["usada_gb"] = float64(usedKB) / 1024 / 1024
					if totalKB > 0 {
						info["percentual_uso"] = float64(usedKB) * 100 / float64(totalKB)
					}
					break
				}
			}
		}
	}

	// If the first method fails, try an alternative approach
	if len(info) == 0 {
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
				freeKB := parseUint64(parts[1])
				usedKB := totalKB - freeKB

				info["total_gb"] = float64(totalKB) / 1024 / 1024
				info["disponivel_gb"] = float64(freeKB) / 1024 / 1024
				info["usada_gb"] = float64(usedKB) / 1024 / 1024
				if totalKB > 0 {
					info["percentual_uso"] = float64(usedKB) * 100 / float64(totalKB)
				}
			}
		}
	}

	// Garantindo valores padrão se não houver dados
	if len(info) == 0 {
		info["total_gb"] = 0.0
		info["disponivel_gb"] = 0.0
		info["usada_gb"] = 0.0
		info["percentual_uso"] = 0.0
	}

	return info
}

// Função para atualizar informações dinâmicas
func updateDynamicInfo(info *SystemInfo) {
	// Atualizando usuários logados
	info.UsuariosLogados = getLoggedUsers()

	// Atualizando informações de processos
	if runtime.GOOS == "windows" {
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
