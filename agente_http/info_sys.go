package main

import (
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

// contains verifica se uma string está presente em um slice de strings
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// getSystemInfo obtém todas as informações do sistema
func getSystemInfo() map[string]interface{} {
	info := make(map[string]interface{})

	// Obter informações do sistema operacional
	sysInfo := getSystemInfoCommand()
	for k, v := range sysInfo {
		info[k] = v
	}

	// Obter informações do usuário atual
	info["usuario_atual"] = getCurrentUser()

	// Obter informações de uptime
	uptimeInfo, uptimeRaw := getUptimeInfo()
	info["uptime"] = uptimeInfo
	info["uptime_raw"] = uptimeRaw

	// Obter informações das impressoras
	info["impressoras"] = getPrinterInfo()

	return info
}

// getSystemInfoCommand executa o comando systeminfo e extrai informações básicas do sistema
func getSystemInfoCommand() map[string]interface{} {
	info := make(map[string]interface{})
	rawSystemInfo := executeSystemInfoCommand()

	// Extrair informações relevantes
	if hostName, ok := rawSystemInfo["Nome do host"]; ok {
		info["nome_host"] = hostName
	}
	if osName, ok := rawSystemInfo["Nome do sistema operacional"]; ok {
		info["nome_so"] = osName
	}
	info["arquitetura"] = runtime.GOARCH

	// Obter versão de compilação
	info["versao_compilacao"] = getOSVersion(rawSystemInfo)

	// Obter informações sobre o último boot
	if lastBoot, ok := rawSystemInfo["Hora da Inicialização do Sistema"]; ok {
		info["ultimo_boot"] = lastBoot
	}

	// Obter informações sobre o fabricante
	if manufacturer, ok := rawSystemInfo["Fabricante do Sistema"]; ok {
		info["fabricante"] = manufacturer
	}

	// Obter informações sobre o modelo
	if model, ok := rawSystemInfo["Modelo do Sistema"]; ok {
		info["modelo"] = model
	}

	// Adicionar informações extras que podem ser úteis
	if biosVersion, ok := rawSystemInfo["Versão do BIOS"]; ok {
		info["versao_bios"] = biosVersion
	}

	return info
}

// executeSystemInfoCommand executa o comando systeminfo e retorna os resultados em um mapa
func executeSystemInfoCommand() map[string]string {
	rawSystemInfo := make(map[string]string)

	// Executar o comando systeminfo
	cmd := exec.Command("systeminfo")
	output, err := cmd.Output()

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
	}

	return rawSystemInfo
}

// getOSVersion obtém a versão do sistema operacional usando vários métodos
func getOSVersion(rawSystemInfo map[string]string) string {
	// Tentar obter do systeminfo primeiro
	if osVersion, ok := rawSystemInfo["Versão do sistema operacional"]; ok {
		return osVersion
	}

	// Método 1: Usando PowerShell para obter a versão do Windows
	cmd := exec.Command("powershell", "-Command", "(Get-CimInstance -class Win32_OperatingSystem).Version")
	output, err := cmd.Output()
	if err == nil && len(output) > 0 {
		return strings.TrimSpace(string(output))
	}

	// Método 2: Usando WMIC como alternativa
	cmd = exec.Command("wmic", "os", "get", "Version", "/value")
	output, err = cmd.Output()
	if err == nil {
		version := strings.TrimSpace(string(output))
		if strings.Contains(version, "=") {
			parts := strings.Split(version, "=")
			if len(parts) >= 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}

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
			return baseVersion + "." + build
		}
		return baseVersion
	}

	// Método 4: Usando o comando ver
	cmd = exec.Command("cmd", "/c", "ver")
	output, err = cmd.Output()
	if err == nil && len(output) > 0 {
		verOutput := strings.TrimSpace(string(output))
		// Extrair a versão usando regex
		re := regexp.MustCompile(`\[Version\s+([0-9\.]+)\]`)
		matches := re.FindStringSubmatch(verOutput)
		if len(matches) >= 2 {
			return matches[1]
		}
	}

	// Último recurso: usar a versão do Go
	return runtime.Version()
}

// getCurrentUser obtém o nome do usuário atual
func getCurrentUser() string {
	cmd := exec.Command("whoami")
	output, err := cmd.Output()
	if err == nil {
		return strings.TrimSpace(string(output))
	}
	return "desconhecido"
}

// getUptimeInfo obtém informações sobre o tempo de atividade do sistema
func getUptimeInfo() (int64, []string) {
	var uptimeMinutes int64
	var uptimeRaw []string

	// Obter uptime em minutos usando PowerShell
	cmd := exec.Command("powershell", "-Command",
		"$uptime = (Get-Date) - (Get-CimInstance -ClassName Win32_OperatingSystem).LastBootUpTime; "+
			"Write-Host $uptime.TotalMinutes")
	output, err := cmd.Output()
	if err == nil {
		uptimeMinutesStr := strings.TrimSpace(string(output))
		// Armazenar no array de uptime_raw
		uptimeRaw = append(uptimeRaw, fmt.Sprintf("powershell_minutes: %s", uptimeMinutesStr))

		// Também armazenar como valor numérico
		var uptimeMinutesFloat float64
		_, err := fmt.Sscanf(uptimeMinutesStr, "%f", &uptimeMinutesFloat)
		if err == nil {
			uptimeMinutes = int64(uptimeMinutesFloat)
		}
	}

	// Obter uptime formatado do PowerShell
	cmd = exec.Command("powershell", "-Command",
		"$uptime = (Get-Date) - (Get-CimInstance -ClassName Win32_OperatingSystem).LastBootUpTime; "+
			"Write-Host \"$($uptime.Days)d $($uptime.Hours)h $($uptime.Minutes)m\"")
	output, err = cmd.Output()
	if err == nil {
		uptimeFormatted := strings.TrimSpace(string(output))
		// Armazenar no array de uptime_raw
		uptimeRaw = append(uptimeRaw, fmt.Sprintf("powershell_formatted: %s", uptimeFormatted))
	}

	return uptimeMinutes, uptimeRaw
}

// getPrinterInfo obtém informações sobre as impressoras instaladas no sistema
func getPrinterInfo() []map[string]interface{} {
	var printers []map[string]interface{}
	var powershellSuccess bool = false

	// Método 1: Usando PowerShell para obter informações detalhadas das impressoras
	cmd := exec.Command("powershell", "-Command",
		"Get-Printer | Select-Object Name, DriverName, PortName, PrinterStatus, Shared, ShareName, Published, Type, DeviceType, Location, Comment | ConvertTo-Json")
	output, err := cmd.Output()

	if err == nil && len(output) > 0 {
		// Armazenar a saída bruta para processamento
		jsonOutput := string(output)

		// Processar a saída JSON para extrair informações estruturadas
		// Como o JSON pode ser um objeto único ou um array, vamos tratar ambos os casos
		if strings.HasPrefix(strings.TrimSpace(jsonOutput), "[") {
			// É um array de impressoras
			cmd = exec.Command("powershell", "-Command",
				"Get-Printer | ForEach-Object { $name = $_.Name; $driver = $_.DriverName; $port = $_.PortName; $status = $_.PrinterStatus; $shared = $_.Shared; $shareName = $_.ShareName; $location = $_.Location; Write-Host \"$name|$driver|$port|$status|$shared|$shareName|$location\" }")
		} else {
			// É uma única impressora
			cmd = exec.Command("powershell", "-Command",
				"$printer = Get-Printer; $name = $printer.Name; $driver = $printer.DriverName; $port = $printer.PortName; $status = $printer.PrinterStatus; $shared = $printer.Shared; $shareName = $printer.ShareName; $location = $printer.Location; Write-Host \"$name|$driver|$port|$status|$shared|$shareName|$location\"")
		}

		printerOutput, err := cmd.Output()
		if err == nil && len(printerOutput) > 0 {
			lines := strings.Split(strings.TrimSpace(string(printerOutput)), "\n")
			for _, line := range lines {
				parts := strings.Split(strings.TrimSpace(line), "|")
				if len(parts) >= 3 {
					printer := make(map[string]interface{})
					printer["nome"] = strings.TrimSpace(parts[0])
					printer["driver"] = strings.TrimSpace(parts[1])
					printer["porta"] = strings.TrimSpace(parts[2])

					if len(parts) >= 4 {
						printer["status"] = strings.TrimSpace(parts[3])
						// Normalizar o status para "Normal" se for "Ready"
						if printer["status"] == "Ready" {
							printer["status"] = "Normal"
						}
					} else {
						printer["status"] = "Normal" // Valor padrão
					}

					if len(parts) >= 5 {
						sharedStr := strings.TrimSpace(parts[4])
						printer["compartilhada"] = (sharedStr == "True")
					} else {
						printer["compartilhada"] = false // Valor padrão
					}

					if len(parts) >= 6 {
						shareName := strings.TrimSpace(parts[5])
						if shareName != "" {
							printer["nome_compartilhamento"] = shareName
						}
					}

					if len(parts) >= 7 {
						location := strings.TrimSpace(parts[6])
						if location != "" {
							printer["localizacao"] = location
						}
					}

					printers = append(printers, printer)
				}
			}

			if len(printers) > 0 {
				powershellSuccess = true
			}
		}
	}

	// Método 2: Usar WMI para obter informações das impressoras se o PowerShell falhar
	if !powershellSuccess {
		cmd := exec.Command("powershell", "-Command",
			"Get-CimInstance -Class Win32_Printer | ForEach-Object { $name = $_.Name; $driver = $_.DriverName; $port = $_.PortName; $status = $_.Status; $shared = $_.Shared; $shareName = $_.ShareName; $location = $_.Location; Write-Host \"$name|$driver|$port|$status|$shared|$shareName|$location\" }")

		output, err := cmd.Output()
		if err == nil && len(output) > 0 {
			lines := strings.Split(strings.TrimSpace(string(output)), "\n")
			for _, line := range lines {
				parts := strings.Split(strings.TrimSpace(line), "|")
				if len(parts) >= 3 {
					printer := make(map[string]interface{})
					printer["nome"] = strings.TrimSpace(parts[0])
					printer["driver"] = strings.TrimSpace(parts[1])
					printer["porta"] = strings.TrimSpace(parts[2])

					if len(parts) >= 4 {
						status := strings.TrimSpace(parts[3])
						// Normalizar o status para "Normal" se for "Idle" ou vazio
						if status == "Idle" || status == "" {
							printer["status"] = "Normal"
						} else {
							printer["status"] = status
						}
					} else {
						printer["status"] = "Normal" // Valor padrão
					}

					if len(parts) >= 5 {
						sharedStr := strings.TrimSpace(parts[4])
						printer["compartilhada"] = (sharedStr == "True")
					} else {
						printer["compartilhada"] = false // Valor padrão
					}

					if len(parts) >= 6 {
						shareName := strings.TrimSpace(parts[5])
						if shareName != "" {
							printer["nome_compartilhamento"] = shareName
						}
					}

					if len(parts) >= 7 {
						location := strings.TrimSpace(parts[6])
						if location != "" {
							printer["localizacao"] = location
						}
					}

					printers = append(printers, printer)
				}
			}
		}

		// Método 3: Usar WMIC diretamente se os métodos anteriores falharem
		if len(printers) == 0 {
			cmd = exec.Command("wmic", "printer", "get", "name,status,drivername,portname,shared,sharename")
			output, err = cmd.Output()

			if err == nil && len(output) > 0 {
				lines := strings.Split(string(output), "\n")
				// Pular a primeira linha (cabeçalho)
				for i := 1; i < len(lines); i++ {
					line := strings.TrimSpace(lines[i])
					if line == "" {
						continue
					}

					// WMIC retorna colunas com largura fixa, precisamos extrair cada campo
					// Formato típico: Name                Status  DriverName                PortName    Shared  ShareName
					fields := strings.Fields(line)
					if len(fields) >= 4 {
						printer := make(map[string]interface{})

						// O nome pode conter espaços, então precisamos determinar onde termina
						// Vamos assumir que o status é uma palavra única (como "OK" ou "Idle")
						nameEndIndex := len(fields) - 5
						if nameEndIndex < 1 {
							nameEndIndex = 1
						}

						name := strings.Join(fields[:nameEndIndex], " ")
						status := fields[nameEndIndex]
						driver := fields[nameEndIndex+1]
						port := fields[nameEndIndex+2]
						sharedStr := fields[nameEndIndex+3]

						printer["nome"] = strings.TrimSpace(name)

						// Normalizar o status
						if status == "OK" || status == "Idle" || status == "" {
							printer["status"] = "Normal"
						} else {
							printer["status"] = strings.TrimSpace(status)
						}

						printer["driver"] = strings.TrimSpace(driver)
						printer["porta"] = strings.TrimSpace(port)
						printer["compartilhada"] = (strings.ToLower(sharedStr) == "true")

						// Adicionar nome de compartilhamento se disponível
						if len(fields) > nameEndIndex+4 {
							shareName := fields[nameEndIndex+4]
							if shareName != "" && strings.ToLower(shareName) != "false" {
								printer["nome_compartilhamento"] = strings.TrimSpace(shareName)
							}
						}

						// Definir trabalhos pendentes como 0 por padrão
						printer["trabalhos_pendentes"] = 0

						printers = append(printers, printer)
					}
				}
			}

			// Se ainda não tiver impressoras, tentar outro formato de WMIC
			if len(printers) == 0 {
				cmd = exec.Command("wmic", "printer", "get", "name,status,drivername,portname,shared,sharename", "/format:csv")
				output, err = cmd.Output()

				if err == nil && len(output) > 0 {
					lines := strings.Split(string(output), "\n")
					// Pular a primeira linha (cabeçalho)
					for i := 1; i < len(lines); i++ {
						line := strings.TrimSpace(lines[i])
						if line == "" || !strings.Contains(line, ",") {
							continue
						}

						parts := strings.Split(line, ",")
						if len(parts) >= 5 {
							printer := make(map[string]interface{})
							// No formato CSV, o primeiro campo geralmente é o nó
							startIndex := 1
							if len(parts) > 5 {
								printer["nome"] = strings.TrimSpace(parts[startIndex])

								status := strings.TrimSpace(parts[startIndex+1])
								if status == "OK" || status == "Idle" || status == "" {
									printer["status"] = "Normal"
								} else {
									printer["status"] = status
								}

								printer["driver"] = strings.TrimSpace(parts[startIndex+2])
								printer["porta"] = strings.TrimSpace(parts[startIndex+3])
								printer["compartilhada"] = (strings.ToLower(strings.TrimSpace(parts[startIndex+4])) == "true")

								if len(parts) > startIndex+5 {
									shareName := strings.TrimSpace(parts[startIndex+5])
									if shareName != "" && strings.ToLower(shareName) != "false" {
										printer["nome_compartilhamento"] = shareName
									}
								}

								printer["trabalhos_pendentes"] = 0
								printers = append(printers, printer)
							}
						}
					}
				}
			}
		}
	}

	// Obter informações adicionais sobre os trabalhos de impressão
	for i, printer := range printers {
		printerName, ok := printer["nome"].(string)
		if ok {
			cmd := exec.Command("powershell", "-Command",
				fmt.Sprintf("Get-PrintJob -PrinterName \"%s\" | Measure-Object | Select-Object -ExpandProperty Count", printerName))
			output, err := cmd.Output()
			if err == nil && len(output) > 0 {
				jobCountStr := strings.TrimSpace(string(output))
				jobCount, err := strconv.Atoi(jobCountStr)
				if err == nil {
					printers[i]["trabalhos_pendentes"] = jobCount
				} else {
					printers[i]["trabalhos_pendentes"] = 0 // Valor padrão
				}
			} else {
				// Método alternativo usando WMI para obter trabalhos pendentes
				cmd = exec.Command("powershell", "-Command",
					fmt.Sprintf("(Get-CimInstance -Class Win32_PrintJob | Where-Object { $_.PrinterName -eq \"%s\" } | Measure-Object).Count", printerName))
				output, err = cmd.Output()
				if err == nil && len(output) > 0 {
					jobCountStr := strings.TrimSpace(string(output))
					jobCount, err := strconv.Atoi(jobCountStr)
					if err == nil {
						printers[i]["trabalhos_pendentes"] = jobCount
					} else {
						printers[i]["trabalhos_pendentes"] = 0 // Valor padrão
					}
				} else {
					printers[i]["trabalhos_pendentes"] = 0 // Valor padrão
				}
			}
		}
	}

	// Garantir que todas as impressoras tenham os campos necessários
	for i := range printers {
		// Garantir que o campo trabalhos_pendentes exista
		if _, ok := printers[i]["trabalhos_pendentes"]; !ok {
			printers[i]["trabalhos_pendentes"] = 0
		}

		// Garantir que o campo status exista
		if _, ok := printers[i]["status"]; !ok {
			printers[i]["status"] = "Normal"
		}

		// Garantir que o campo compartilhada exista
		if _, ok := printers[i]["compartilhada"]; !ok {
			printers[i]["compartilhada"] = false
		}
	}

	return printers
}

func getLoggedUsers() []string {
	var users []string

	// Obter usuários logados usando PowerShell
	cmd := exec.Command("powershell", "-Command", "Get-CimInstance -Class Win32_ComputerSystem | Select-Object -ExpandProperty UserName")
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
