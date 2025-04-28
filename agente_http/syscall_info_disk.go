package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"syscall"
	"unsafe"
)

func getDiskInfoSyscall() map[string]interface{} {
	info := make(map[string]interface{})

	// Inicializar as DLLs e procedimentos
	err := initWindowsDLLs()
	if err != nil {
		info["erro"] = err.Error()
		return info
	}

	// Verificar se temos os procedimentos necessários
	if getLogicalDrivesFn == nil || getDiskFreeSpaceExFn == nil || getVolumeInformationFn == nil {
		info["erro"] = "Não foi possível carregar funções de disco"
		return info
	}

	// Primeiro, obter informações detalhadas dos discos físicos usando Get-CimInstance e Get-PhysicalDisk
	diskModels := make(map[string]map[string]interface{})

	// Comando direto para obter informações do disco físico
	diskInfoOutput, err := executeCommand("powershell", "-Command",
		"[Console]::OutputEncoding = [System.Text.Encoding]::UTF8; "+
			"$disk = Get-CimInstance -ClassName Win32_DiskDrive | Select-Object DeviceID, Model, Manufacturer, SerialNumber; "+
			"$disk | ConvertTo-Json")

	// Verificar se o comando PowerShell falhou
	if err != nil || strings.TrimSpace(diskInfoOutput) == "" {
		// Tentar com WMIC se o PowerShell falhar
		diskInfoOutput, _ = executeCommand("cmd", "/c", "wmic diskdrive get DeviceID, Model, SerialNumber /format:csv")

		// Processar saída do WMIC
		lines := strings.Split(strings.TrimSpace(diskInfoOutput), "\n")
		if len(lines) > 1 {
			// Primeira linha é o cabeçalho
			header := strings.Split(lines[0], ",")

			// Mapear índices das colunas
			deviceIdIdx := -1
			modelIdx := -1
			serialIdx := -1

			for i, col := range header {
				col = strings.TrimSpace(col)
				switch col {
				case "DeviceID":
					deviceIdIdx = i
				case "Model":
					modelIdx = i
				case "SerialNumber":
					serialIdx = i
				}
			}

			// Processar cada linha
			for i := 1; i < len(lines); i++ {
				line := strings.TrimSpace(lines[i])
				if line == "" {
					continue
				}

				fields := strings.Split(line, ",")
				if len(fields) <= 1 {
					continue
				}

				if deviceIdIdx >= 0 && deviceIdIdx < len(fields) {
					deviceId := strings.TrimSpace(fields[deviceIdIdx])
					diskInfo := make(map[string]interface{})

					if modelIdx >= 0 && modelIdx < len(fields) {
						diskInfo["modelo"] = strings.TrimSpace(fields[modelIdx])
						diskInfo["nome_amigavel"] = strings.TrimSpace(fields[modelIdx])
					}

					if serialIdx >= 0 && serialIdx < len(fields) {
						diskInfo["numero_serie"] = strings.TrimSpace(fields[serialIdx])
					}

					// Inicializar array de letras
					diskInfo["letras"] = make([]map[string]interface{}, 0)

					diskModels[deviceId] = diskInfo
				}
			}
		}
	} else {
		// Tentar desserializar como um único objeto primeiro
		var singleDisk map[string]interface{}
		singleErr := json.Unmarshal([]byte(diskInfoOutput), &singleDisk)

		if singleErr == nil && len(singleDisk) > 0 {
			// É um único disco
			deviceId, ok := singleDisk["DeviceID"].(string)
			if ok {
				diskInfo := make(map[string]interface{})

				if model, ok := singleDisk["Model"].(string); ok && model != "" {
					diskInfo["modelo"] = strings.TrimSpace(model)
					diskInfo["nome_amigavel"] = strings.TrimSpace(model)
				}

				if serial, ok := singleDisk["SerialNumber"].(string); ok && serial != "" {
					diskInfo["numero_serie"] = strings.TrimSpace(serial)
				}

				// Inicializar array de letras
				diskInfo["letras"] = make([]map[string]interface{}, 0)

				diskModels[deviceId] = diskInfo
			}
		} else {
			// Tentar como array
			var diskArray []map[string]interface{}
			arrayErr := json.Unmarshal([]byte(diskInfoOutput), &diskArray)

			if arrayErr == nil && len(diskArray) > 0 {
				for _, disk := range diskArray {
					deviceId, ok := disk["DeviceID"].(string)
					if ok {
						diskInfo := make(map[string]interface{})

						if model, ok := disk["Model"].(string); ok && model != "" {
							diskInfo["modelo"] = strings.TrimSpace(model)
							diskInfo["nome_amigavel"] = strings.TrimSpace(model)
						}

						if serial, ok := disk["SerialNumber"].(string); ok && serial != "" {
							diskInfo["numero_serie"] = strings.TrimSpace(serial)
						}

						// Inicializar array de letras
						diskInfo["letras"] = make([]map[string]interface{}, 0)

						diskModels[deviceId] = diskInfo
					}
				}
			}
		}
	}

	// Se não conseguiu obter informações via JSON ou WMIC, tentar método alternativo
	if len(diskModels) == 0 {
		// Método alternativo usando formato de texto
		diskInfoText, _ := executeCommand("powershell", "-Command",
			"[Console]::OutputEncoding = [System.Text.Encoding]::UTF8; "+
				"Get-CimInstance -ClassName Win32_DiskDrive | ForEach-Object { "+
				"  Write-Host $_.DeviceID '|' $_.Model '|' $_.Manufacturer '|' $_.SerialNumber "+
				"}")

		// Se o PowerShell falhar, tentar com WMIC
		if diskInfoText == "" {
			diskInfoText, _ = executeCommand("cmd", "/c", "wmic diskdrive get DeviceID, Model, SerialNumber")
			// Processar saída do WMIC em formato de tabela
			lines := strings.Split(strings.TrimSpace(diskInfoText), "\n")
			if len(lines) > 1 {
				// Primeira linha é o cabeçalho
				for i := 1; i < len(lines); i++ {
					line := strings.TrimSpace(lines[i])
					if line == "" {
						continue
					}

					// Dividir por espaços, mas preservar múltiplos espaços como um único separador
					fields := strings.Fields(line)
					if len(fields) >= 2 {
						deviceId := fields[0]
						model := strings.Join(fields[1:len(fields)-1], " ")
						serial := fields[len(fields)-1]

						diskInfo := make(map[string]interface{})
						diskInfo["modelo"] = model
						diskInfo["nome_amigavel"] = model
						diskInfo["numero_serie"] = serial

						// Inicializar array de letras
						diskInfo["letras"] = make([]map[string]interface{}, 0)

						diskModels[deviceId] = diskInfo
					}
				}
			}
		} else {
			lines := strings.Split(strings.TrimSpace(diskInfoText), "\n")
			for _, line := range lines {
				parts := strings.Split(line, "|")
				if len(parts) >= 4 {
					deviceId := strings.TrimSpace(parts[0])
					model := strings.TrimSpace(parts[1])
					serial := strings.TrimSpace(parts[3])

					diskInfo := make(map[string]interface{})
					if model != "" {
						diskInfo["modelo"] = model
						diskInfo["nome_amigavel"] = model
					}
					if serial != "" {
						diskInfo["numero_serie"] = serial
					}

					// Inicializar array de letras
					diskInfo["letras"] = make([]map[string]interface{}, 0)

					diskModels[deviceId] = diskInfo
				}
			}
		}
	}

	// Método direto para obter informações do PhysicalDisk
	physicalDiskOutput, err := executeCommand("powershell", "-Command",
		"[Console]::OutputEncoding = [System.Text.Encoding]::UTF8; "+
			"Get-PhysicalDisk | ForEach-Object { "+
			"  Write-Host $_.DeviceId '|' $_.FriendlyName '|' $_.SerialNumber '|' $_.FirmwareVersion '|' $_.MediaType '|' $_.BusType '|' $_.HealthStatus '|' $_.OperationalStatus '|' $_.FruId "+
			"}")

	// Se o PowerShell falhar, tentar com WMIC para obter informações adicionais
	if err != nil || strings.TrimSpace(physicalDiskOutput) == "" {
		// WMIC não tem um comando direto equivalente ao Get-PhysicalDisk, mas podemos obter algumas informações
		mediaTypeOutput, _ := executeCommand("cmd", "/c", "wmic diskdrive get DeviceID, MediaType, InterfaceType, Status /format:csv")

		// Processar saída do WMIC
		lines := strings.Split(strings.TrimSpace(mediaTypeOutput), "\n")
		if len(lines) > 1 {
			// Primeira linha é o cabeçalho
			header := strings.Split(lines[0], ",")

			// Mapear índices das colunas
			deviceIdIdx := -1
			mediaTypeIdx := -1
			interfaceTypeIdx := -1
			statusIdx := -1

			for i, col := range header {
				col = strings.TrimSpace(col)
				switch col {
				case "DeviceID":
					deviceIdIdx = i
				case "MediaType":
					mediaTypeIdx = i
				case "InterfaceType":
					interfaceTypeIdx = i
				case "Status":
					statusIdx = i
				}
			}

			// Processar cada linha
			for i := 1; i < len(lines); i++ {
				line := strings.TrimSpace(lines[i])
				if line == "" {
					continue
				}

				fields := strings.Split(line, ",")
				if len(fields) <= 1 {
					continue
				}

				if deviceIdIdx >= 0 && deviceIdIdx < len(fields) {
					deviceId := strings.TrimSpace(fields[deviceIdIdx])

					// Verificar se já temos informações para este disco
					diskInfo, exists := diskModels[deviceId]
					if !exists {
						diskInfo = make(map[string]interface{})
						diskInfo["letras"] = make([]map[string]interface{}, 0)
						diskModels[deviceId] = diskInfo
					}

					// Adicionar ou atualizar informações
					if mediaTypeIdx >= 0 && mediaTypeIdx < len(fields) {
						mediaType := strings.TrimSpace(fields[mediaTypeIdx])
						if mediaType != "" {
							diskInfo["tipo_midia"] = mediaType
						}
					}

					if interfaceTypeIdx >= 0 && interfaceTypeIdx < len(fields) {
						interfaceType := strings.TrimSpace(fields[interfaceTypeIdx])
						if interfaceType != "" {
							diskInfo["tipo_barramento"] = interfaceType
						}
					}

					if statusIdx >= 0 && statusIdx < len(fields) {
						status := strings.TrimSpace(fields[statusIdx])
						if status != "" {
							diskInfo["status_operacional"] = status
							// Mapear status para um valor de saúde
							if status == "OK" {
								diskInfo["status_saude"] = "Healthy"
							} else {
								diskInfo["status_saude"] = "Unhealthy"
							}
						}
					}
				}
			}
		}

		// Tentar obter informações de firmware
		firmwareOutput, _ := executeCommand("cmd", "/c", "wmic diskdrive get DeviceID, FirmwareRevision /format:csv")

		// Processar saída do WMIC
		lines = strings.Split(strings.TrimSpace(firmwareOutput), "\n")
		if len(lines) > 1 {
			// Primeira linha é o cabeçalho
			header := strings.Split(lines[0], ",")

			// Mapear índices das colunas
			deviceIdIdx := -1
			firmwareIdx := -1

			for i, col := range header {
				col = strings.TrimSpace(col)
				switch col {
				case "DeviceID":
					deviceIdIdx = i
				case "FirmwareRevision":
					firmwareIdx = i
				}
			}

			// Processar cada linha
			for i := 1; i < len(lines); i++ {
				line := strings.TrimSpace(lines[i])
				if line == "" {
					continue
				}

				fields := strings.Split(line, ",")
				if len(fields) <= 1 {
					continue
				}

				if deviceIdIdx >= 0 && deviceIdIdx < len(fields) && firmwareIdx >= 0 && firmwareIdx < len(fields) {
					deviceId := strings.TrimSpace(fields[deviceIdIdx])
					firmware := strings.TrimSpace(fields[firmwareIdx])

					// Verificar se já temos informações para este disco
					diskInfo, exists := diskModels[deviceId]
					if exists && firmware != "" {
						diskInfo["versao_firmware"] = firmware
					}
				}
			}
		}
	} else {
		// Processar saída do PhysicalDisk
		physicalDiskLines := strings.Split(strings.TrimSpace(physicalDiskOutput), "\n")
		for _, line := range physicalDiskLines {
			parts := strings.Split(line, "|")
			if len(parts) >= 9 {
				deviceIdStr := strings.TrimSpace(parts[0])
				friendlyName := strings.TrimSpace(parts[1])
				serialNumber := strings.TrimSpace(parts[2])
				firmwareVersion := strings.TrimSpace(parts[3])
				mediaType := strings.TrimSpace(parts[4])
				busType := strings.TrimSpace(parts[5])
				healthStatus := strings.TrimSpace(parts[6])
				operationalStatus := strings.TrimSpace(parts[7])
				fruId := strings.TrimSpace(parts[8])

				// Converter para o formato do Win32_DiskDrive
				deviceId := "\\\\.\\PHYSICALDRIVE" + deviceIdStr

				// Verificar se já temos informações para este disco
				diskInfo, exists := diskModels[deviceId]
				if !exists {
					diskInfo = make(map[string]interface{})
					diskInfo["letras"] = make([]map[string]interface{}, 0)
					diskModels[deviceId] = diskInfo
				}

				// Adicionar ou atualizar informações
				if friendlyName != "" {
					diskInfo["nome_amigavel"] = friendlyName
				}
				if serialNumber != "" {
					diskInfo["numero_serie"] = serialNumber
				}
				if firmwareVersion != "" {
					diskInfo["versao_firmware"] = firmwareVersion
				}
				if mediaType != "" {
					diskInfo["tipo_midia"] = mediaType
				}
				if busType != "" {
					diskInfo["tipo_barramento"] = busType
				}
				if healthStatus != "" {
					diskInfo["status_saude"] = healthStatus
				}
				if operationalStatus != "" {
					diskInfo["status_operacional"] = operationalStatus
				}
				if fruId != "" {
					diskInfo["numero_serie"] = fruId // Substituir pelo FruId se disponível
				}
			}
		}
	}

	// Agora, obter mapeamento entre discos físicos e letras de unidade
	diskToLetter := make(map[string][]string)

	// Método direto para obter mapeamento
	mapOutput, err := executeCommand("powershell", "-Command",
		"[Console]::OutputEncoding = [System.Text.Encoding]::UTF8; "+
			"Get-CimInstance -Class Win32_LogicalDisk | ForEach-Object { "+
			"  $disk = $_; "+
			"  $diskLetter = $disk.DeviceID; "+
			"  $partitions = Get-CimInstance -Query \"ASSOCIATORS OF {$disk} WHERE ResultClass = Win32_DiskPartition\"; "+
			"  foreach ($partition in $partitions) { "+
			"    $drives = Get-CimInstance -Query \"ASSOCIATORS OF {$partition} WHERE ResultClass = Win32_DiskDrive\"; "+
			"    foreach ($drive in $drives) { "+
			"      Write-Host \"$diskLetter|$($drive.DeviceID)\" "+
			"    } "+
			"  } "+
			"}")

	// Se o PowerShell falhar, tentar com WMIC
	if err != nil || strings.TrimSpace(mapOutput) == "" {
		// Usar WMIC para obter mapeamento entre discos físicos e letras
		// Primeiro, obter informações das partições
		partitionOutput, _ := executeCommand("cmd", "/c", "wmic partition get DiskIndex, DeviceID /format:csv")

		// Criar mapa de partição para disco
		partitionToDisk := make(map[string]string)

		lines := strings.Split(strings.TrimSpace(partitionOutput), "\n")
		if len(lines) > 1 {
			// Primeira linha é o cabeçalho
			header := strings.Split(lines[0], ",")

			// Mapear índices das colunas
			diskIndexIdx := -1
			deviceIdIdx := -1

			for i, col := range header {
				col = strings.TrimSpace(col)
				switch col {
				case "DiskIndex":
					diskIndexIdx = i
				case "DeviceID":
					deviceIdIdx = i
				}
			}

			// Processar cada linha
			for i := 1; i < len(lines); i++ {
				line := strings.TrimSpace(lines[i])
				if line == "" {
					continue
				}

				fields := strings.Split(line, ",")
				if len(fields) <= 1 {
					continue
				}

				if diskIndexIdx >= 0 && diskIndexIdx < len(fields) && deviceIdIdx >= 0 && deviceIdIdx < len(fields) {
					diskIndex := strings.TrimSpace(fields[diskIndexIdx])
					partitionId := strings.TrimSpace(fields[deviceIdIdx])

					if diskIndex != "" && partitionId != "" {
						partitionToDisk[partitionId] = "\\\\.\\PHYSICALDRIVE" + diskIndex
					}
				}
			}
		}

		// Agora, obter mapeamento entre volumes lógicos e partições
		volumeOutput, _ := executeCommand("cmd", "/c", "wmic volume get DriveLetter, DeviceID /format:csv")

		lines = strings.Split(strings.TrimSpace(volumeOutput), "\n")
		if len(lines) > 1 {
			// Primeira linha é o cabeçalho
			header := strings.Split(lines[0], ",")

			// Mapear índices das colunas
			driveLetterIdx := -1
			deviceIdIdx := -1

			for i, col := range header {
				col = strings.TrimSpace(col)
				switch col {
				case "DriveLetter":
					driveLetterIdx = i
				case "DeviceID":
					deviceIdIdx = i
				}
			}

			// Processar cada linha
			for i := 1; i < len(lines); i++ {
				line := strings.TrimSpace(lines[i])
				if line == "" {
					continue
				}

				fields := strings.Split(line, ",")
				if len(fields) <= 1 {
					continue
				}

				if driveLetterIdx >= 0 && driveLetterIdx < len(fields) && deviceIdIdx >= 0 && deviceIdIdx < len(fields) {
					driveLetter := strings.TrimSpace(fields[driveLetterIdx])
					volumeId := strings.TrimSpace(fields[deviceIdIdx])

					if driveLetter != "" && volumeId != "" {
						// Tentar encontrar a partição correspondente a este volume
						for partitionId, diskId := range partitionToDisk {
							if strings.Contains(volumeId, partitionId) {
								// Adicionar letra ao array de letras do disco
								if _, exists := diskToLetter[diskId]; !exists {
									diskToLetter[diskId] = make([]string, 0)
								}
								diskToLetter[diskId] = append(diskToLetter[diskId], driveLetter)
								break
							}
						}
					}
				}
			}
		}
	} else {
		lines := strings.Split(strings.TrimSpace(mapOutput), "\n")
		for _, line := range lines {
			parts := strings.Split(line, "|")
			if len(parts) == 2 {
				letter := strings.TrimSpace(parts[0])
				diskId := strings.TrimSpace(parts[1])

				// Adicionar letra ao array de letras do disco
				if _, exists := diskToLetter[diskId]; !exists {
					diskToLetter[diskId] = make([]string, 0)
				}
				diskToLetter[diskId] = append(diskToLetter[diskId], letter)
			}
		}
	}

	// Se não conseguiu mapear, usar método alternativo
	if len(diskToLetter) == 0 {
		// Método alternativo: associar todas as letras ao primeiro disco físico
		for deviceId := range diskModels {
			diskToLetter[deviceId] = []string{}

			// Obter as letras de unidade disponíveis
			drives, _, _ := getLogicalDrivesFn.Call()

			// Iterar sobre as letras de unidade (A-Z)
			for i := 0; i < 26; i++ {
				if drives&(1<<i) == 0 {
					continue
				}

				driveLetter := string('A' + i)
				diskToLetter[deviceId] = append(diskToLetter[deviceId], driveLetter+":")
			}

			break // Usar apenas o primeiro disco
		}
	}

	// Obter as letras de unidade disponíveis
	drives, _, _ := getLogicalDrivesFn.Call()

	// Coletar informações de cada letra de unidade
	letterInfos := make(map[string]map[string]interface{})

	// Iterar sobre as letras de unidade (A-Z)
	for i := 0; i < 26; i++ {
		if drives&(1<<i) == 0 {
			continue
		}

		driveLetter := string('A' + i)
		rootPath := driveLetter + ":\\"
		rootPathPtr, _ := syscall.UTF16PtrFromString(rootPath)

		// Obter informações de espaço em disco
		var totalBytes uint64
		ret, _, _ := getDiskFreeSpaceExFn.Call(
			uintptr(unsafe.Pointer(rootPathPtr)),
			0,
			uintptr(unsafe.Pointer(&totalBytes)),
			0,
		)

		if ret == 0 {
			continue // Falha ao obter informações
		}

		// Obter informações de volume
		volumeNameBuffer := make([]uint16, 256)
		fileSystemNameBuffer := make([]uint16, 256)
		var volumeSerialNumber uint32
		var maximumComponentLength uint32
		var fileSystemFlags uint32

		ret, _, _ = getVolumeInformationFn.Call(
			uintptr(unsafe.Pointer(rootPathPtr)),
			uintptr(unsafe.Pointer(&volumeNameBuffer[0])),
			uintptr(len(volumeNameBuffer)),
			uintptr(unsafe.Pointer(&volumeSerialNumber)),
			uintptr(unsafe.Pointer(&maximumComponentLength)),
			uintptr(unsafe.Pointer(&fileSystemFlags)),
			uintptr(unsafe.Pointer(&fileSystemNameBuffer[0])),
			uintptr(len(fileSystemNameBuffer)),
		)

		volumeName := "Sem Rótulo"
		fileSystemName := "Desconhecido"

		if ret != 0 {
			volumeName = syscall.UTF16ToString(volumeNameBuffer)
			fileSystemName = syscall.UTF16ToString(fileSystemNameBuffer)
		}

		// Adicionar informações da letra
		letterInfo := make(map[string]interface{})
		letterInfo["letra"] = driveLetter
		letterInfo["rotulo"] = volumeName
		letterInfo["sistema_arquivos"] = fileSystemName
		letterInfo["tamanho_total"] = totalBytes

		// Adicionar número de série do volume
		if volumeSerialNumber > 0 {
			letterInfo["numero_serie_volume"] = formatVolumeSerial(volumeSerialNumber)
		}

		letterInfos[driveLetter+":"] = letterInfo
	}

	// Agora, associar as informações das letras aos discos físicos
	discos := make([]map[string]interface{}, 0)

	for deviceId, diskInfo := range diskModels {
		// Verificar se temos letras associadas a este disco
		if letters, ok := diskToLetter[deviceId]; ok {
			// Adicionar cada letra ao array de letras do disco
			letrasArray := diskInfo["letras"].([]map[string]interface{})

			for _, letter := range letters {
				if letterInfo, ok := letterInfos[letter]; ok {
					letrasArray = append(letrasArray, letterInfo)
				}
			}

			diskInfo["letras"] = letrasArray
		}

		discos = append(discos, diskInfo)
	}

	// Se não temos discos mapeados, criar um disco genérico com todas as letras
	if len(discos) == 0 {
		diskGenerico := make(map[string]interface{})
		diskGenerico["modelo"] = "Disco Desconhecido"

		letras := make([]map[string]interface{}, 0)
		for _, letterInfo := range letterInfos {
			letras = append(letras, letterInfo)
		}

		diskGenerico["letras"] = letras
		discos = append(discos, diskGenerico)
	}

	info["discos"] = discos

	return info
}

// Formata o número de série do volume no formato padrão (XXXX-XXXX)
func formatVolumeSerial(serial uint32) string {
	return fmt.Sprintf("%04X-%04X", (serial>>16)&0xFFFF, serial&0xFFFF)
}

// Normaliza texto para garantir que caracteres especiais sejam exibidos corretamente
func normalizarTexto(texto string) string {
	// Substituições comuns para caracteres especiais em português
	replacements := map[string]string{
		"padr�o": "padrão",
	}

	result := texto
	for incorrect, correct := range replacements {
		result = strings.Replace(result, incorrect, correct, -1)
	}

	return result
}

// Função auxiliar para obter as chaves de um mapa
func getKeys(m map[string]map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
