package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"unsafe"
)

// Estrutura para informações do sistema via syscall
type systemInfo struct {
	dwOemID                     uint32
	dwPageSize                  uint32
	lpMinimumApplicationAddress uintptr
	lpMaximumApplicationAddress uintptr
	dwActiveProcessorMask       uintptr
	dwNumberOfProcessors        uint32
	dwProcessorType             uint32
	dwAllocationGranularity     uint32
	wProcessorLevel             uint16
	wProcessorRevision          uint16
}

// Estrutura para informações de versão do sistema operacional
type osVersionInfoEx struct {
	dwOSVersionInfoSize uint32
	dwMajorVersion      uint32
	dwMinorVersion      uint32
	dwBuildNumber       uint32
	dwPlatformId        uint32
	szCSDVersion        [128]uint16
	wServicePackMajor   uint16
	wServicePackMinor   uint16
	wSuiteMask          uint16
	wProductType        byte
	wReserved           byte
}

// Estrutura para informações de tempo do sistema
type systemTime struct {
	wYear         uint16
	wMonth        uint16
	wDay          uint16
	wDayOfWeek    uint16
	wHour         uint16
	wMinute       uint16
	wSecond       uint16
	wMilliseconds uint16
}

// Função centralizada para obter informações do sistema
func getSystemInfoData() (systemInfo, error) {
	var sysInfo systemInfo

	// Inicializar as DLLs e procedimentos
	err := initWindowsDLLs()
	if err != nil {
		return sysInfo, err
	}

	// Chamar a função
	getSystemInfoFn.Call(uintptr(unsafe.Pointer(&sysInfo)))

	return sysInfo, nil
}

// Mantém a mesma interface da função original em syscall_info_cpu.go
func getCPUInfoSyscall() map[string]interface{} {
	info := make(map[string]interface{})

	// Inicializar as DLLs e procedimentos
	err := initWindowsDLLs()
	if err != nil {
		info["erro"] = err.Error()
		return info
	}

	// Obter informações do sistema
	sysInfo, err := getSystemInfoData()
	if err != nil {
		info["erro"] = err.Error()
		return info
	}

	// Determinar a arquitetura do processador
	switch sysInfo.dwOemID >> 8 {
	case 0:
		info["arquitetura"] = "x32"
	case 9:
		info["arquitetura"] = "x64"
	case 5:
		info["arquitetura"] = "ARM"
	case 12:
		info["arquitetura"] = "ARM64"
	default:
		info["arquitetura"] = fmt.Sprintf("Desconhecida (%d)", sysInfo.dwOemID>>8)
	}

	// Preencher o mapa de informações
	info["nucleos"] = int(sysInfo.dwNumberOfProcessors)
	info["tipo_processador"] = int(sysInfo.dwProcessorType)
	info["nivel_processador"] = int(sysInfo.wProcessorLevel)
	info["revisao_processador"] = int(sysInfo.wProcessorRevision)

	// Obter informações adicionais do processador
	processorInfo := getProcessorInfoSyscall()
	for k, v := range processorInfo {
		info[k] = v
	}

	return info
}

// Mantém a mesma interface da função original em syscall_info_hw.go
func getHardwareInfoSyscall() map[string]interface{} {
	info := make(map[string]interface{})

	// Inicializar as DLLs e procedimentos
	err := initWindowsDLLs()
	if err != nil {
		info["erro"] = err.Error()
		return info
	}

	// Tentar obter informações do computador via registro
	if regOpenKeyExFn != nil && regQueryValueExFn != nil && regCloseKeyFn != nil {
		// Constantes para o registro do Windows
		const HKEY_LOCAL_MACHINE = 0x80000002
		const KEY_READ = 0x20019

		// Caminho para informações do sistema
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

			// Obter fabricante
			info["fabricante"] = getRegistryString(hKey, "SystemManufacturer")

			// Obter modelo
			info["modelo"] = getRegistryString(hKey, "SystemProductName")

			// Obter versão da BIOS
			info["versao_bios"] = getRegistryString(hKey, "BIOSVersion")

			// Obter data da BIOS
			info["data_bios"] = getRegistryString(hKey, "BIOSReleaseDate")
            
            // Obter número de série do sistema
            info["numero_serie"] = getRegistryString(hKey, "SerialNumber")
		}
	}

	// Se não conseguiu obter o número de série via registro, tentar via PowerShell
	if _, ok := info["numero_serie"]; !ok || info["numero_serie"] == "Desconhecido" {
		// Método 1: Usar PowerShell com Get-CimInstance
		serialNumber, err := executeCommand("powershell", "-Command", "(Get-CimInstance -ClassName Win32_BIOS).SerialNumber")
		if err == nil && strings.TrimSpace(serialNumber) != "" {
			info["numero_serie"] = strings.TrimSpace(serialNumber)
		} else {
			// Método 2: Usar WMIC como alternativa
			serialNumber, err = executeCommand("wmic", "bios", "get", "SerialNumber")
			if err == nil && strings.TrimSpace(serialNumber) != "" {
				// Remover o cabeçalho "SerialNumber" da saída do WMIC
				lines := strings.Split(strings.TrimSpace(serialNumber), "\n")
				if len(lines) > 1 {
					info["numero_serie"] = strings.TrimSpace(lines[1])
				} else if len(lines) == 1 && !strings.Contains(strings.ToLower(lines[0]), "serialnumber") {
					info["numero_serie"] = strings.TrimSpace(lines[0])
				}
			}
		}
	}

	// Adicionar informações básicas se não foram obtidas
	if _, ok := info["fabricante"]; !ok {
		info["fabricante"] = "Desconhecido"
	}
	if _, ok := info["modelo"]; !ok {
		info["modelo"] = "Desconhecido"
	}
	if _, ok := info["numero_serie"]; !ok {
		info["numero_serie"] = "Desconhecido"
	}

	return info
}

// Implementação da função getSystemArchitecture de syscall_info_sys.go
func getSystemArchitecture() string {
	// Inicializar as DLLs e procedimentos
	err := initWindowsDLLs()
	if err != nil {
		return "Desconhecido"
	}

	// Verificar diretamente as variáveis de ambiente primeiro (método mais confiável)
	if os.Getenv("PROCESSOR_ARCHITECTURE") == "AMD64" ||
		os.Getenv("PROCESSOR_ARCHITEW6432") == "AMD64" {
		return "x64"
	}

	// Primeiro, tentar usar GetNativeSystemInfo para obter a arquitetura real do sistema
	if getNativeSystemInfoFn != nil {
		var sysInfo systemInfo
		getNativeSystemInfoFn.Call(uintptr(unsafe.Pointer(&sysInfo)))

		// Verificar a arquitetura do processador
		processorArch := sysInfo.dwOemID >> 8
		if processorArch == 9 || processorArch == 12 {
			return "x64"
		} else if processorArch == 5 {
			return "ARM"
		} else if processorArch == 6 {
			return "ARM64"
		}
		// Se não for nenhum desses, continuar com outros métodos
	}

	// Verificar se temos os procedimentos necessários para IsWow64Process
	if isWow64ProcessFn != nil && getCurrentProcessFn != nil {
		// Obter o handle do processo atual
		handle, _, _ := getCurrentProcessFn.Call()

		var isWow64 bool
		ret, _, _ := isWow64ProcessFn.Call(handle, uintptr(unsafe.Pointer(&isWow64)))

		if ret != 0 && isWow64 {
			return "x64" // Processo 32 bits rodando em sistema 64 bits
		}
	}

	// Tentar obter informações do sistema usando GetSystemInfo como último recurso
	sysInfo, err := getSystemInfoData()
	if err == nil {
		// Verificar a arquitetura do processador
		processorArch := sysInfo.dwOemID >> 8
		if processorArch == 9 || processorArch == 12 {
			return "x64"
		} else if processorArch == 5 {
			return "ARM"
		} else if processorArch == 6 {
			return "ARM64"
		}
	}

	// Verificar se estamos em um processo de 64 bits
	// Em Go, unsafe.Sizeof(uintptr(0)) retorna 8 em sistemas de 64 bits e 4 em sistemas de 32 bits
	if unsafe.Sizeof(uintptr(0)) == 8 {
		return "x64"
	}

	// Se tudo falhar, tentar executar um comando para verificar a arquitetura
	output, err := executeCommand("wmic", "OS", "get", "OSArchitecture")
	if err == nil && strings.Contains(output, "64") {
		return "x64"
	}

	return "x32" // Padrão para sistemas x32
}

// Mantém a mesma interface da função getProcessorInfoSyscall de syscall_info_cpu.go
func getProcessorInfoSyscall() map[string]interface{} {
	info := make(map[string]interface{})

	// Já inicializamos as DLLs em initWindowsDLLs
	if !dllsInitialized {
		err := initWindowsDLLs()
		if err != nil {
			return info
		}
	}

	// Tentar obter informações detalhadas do processador via registro
	if regOpenKeyExFn != nil && regQueryValueExFn != nil && regCloseKeyFn != nil {
		// Constantes para o registro do Windows
		const HKEY_LOCAL_MACHINE = 0x80000002
		const KEY_READ = 0x20019

		// Caminho para informações do processador
		keyPath, _ := syscall.UTF16PtrFromString("HARDWARE\\DESCRIPTION\\System\\CentralProcessor\\0")

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

			// Obter nome/modelo do processador
			modeloProcessador := getRegistryString(hKey, "ProcessorNameString")
			// Remover espaços em branco extras no início e no final
			info["modelo"] = strings.TrimSpace(modeloProcessador)

			// Obter identificador do processador
			info["identificador"] = getRegistryString(hKey, "Identifier")

			// Obter fabricante do processador
			info["fabricante"] = getRegistryString(hKey, "VendorIdentifier")

			// Obter frequência do processador em MHz
			var mhz uint32
			mhzValueName, _ := syscall.UTF16PtrFromString("~MHz")
			var dataType uint32
			var dataSize uint32 = 4

			ret, _, _ = regQueryValueExFn.Call(
				uintptr(hKey),
				uintptr(unsafe.Pointer(mhzValueName)),
				0,
				uintptr(unsafe.Pointer(&dataType)),
				uintptr(unsafe.Pointer(&mhz)),
				uintptr(unsafe.Pointer(&dataSize)),
			)

			if ret == 0 {
				info["frequencia_mhz"] = int(mhz)
				info["frequencia"] = fmt.Sprintf("%.2f GHz", float64(mhz)/1000.0)
			}

			// Obter informações sobre cache L2 e L3
			var cacheSize uint32
			cacheValueName, _ := syscall.UTF16PtrFromString("L2CacheSize")
			dataSize = 4

			ret, _, _ = regQueryValueExFn.Call(
				uintptr(hKey),
				uintptr(unsafe.Pointer(cacheValueName)),
				0,
				uintptr(unsafe.Pointer(&dataType)),
				uintptr(unsafe.Pointer(&cacheSize)),
				uintptr(unsafe.Pointer(&dataSize)),
			)

			if ret == 0 {
				info["cache_l2"] = fmt.Sprintf("%d KB", cacheSize)
			}

			cacheValueName, _ = syscall.UTF16PtrFromString("L3CacheSize")
			dataSize = 4

			ret, _, _ = regQueryValueExFn.Call(
				uintptr(hKey),
				uintptr(unsafe.Pointer(cacheValueName)),
				0,
				uintptr(unsafe.Pointer(&dataType)),
				uintptr(unsafe.Pointer(&cacheSize)),
				uintptr(unsafe.Pointer(&dataSize)),
			)

			if ret == 0 {
				info["cache_l3"] = fmt.Sprintf("%d KB", cacheSize)
			}
		}
	}

	// Adicionar informações básicas se não foram obtidas
	if _, ok := info["modelo"]; !ok {
		info["modelo"] = "Desconhecido"
	}
	if _, ok := info["fabricante"]; !ok {
		info["fabricante"] = "Desconhecido"
	}

	return info
}

func getSystemInfoSyscall() map[string]interface{} {
	info := make(map[string]interface{})

	// Inicializar as DLLs e procedimentos
	err := initWindowsDLLs()
	if err != nil {
		info["erro"] = err.Error()
		return info
	}

	// Obter nome do host
	if getComputerNameExFn != nil {
		var size uint32 = 260 // MAX_COMPUTERNAME_LENGTH + 1
		var buffer [260]uint16

		ret, _, _ := getComputerNameExFn.Call(
			uintptr(ComputerNameDnsHostname),
			uintptr(unsafe.Pointer(&buffer[0])),
			uintptr(unsafe.Pointer(&size)),
		)

		if ret != 0 {
			info["nome_host"] = syscall.UTF16ToString(buffer[:size])
		}
	}

	// Obter versão do sistema operacional
	if rtlGetVersionFn != nil {
		var osInfo osVersionInfoEx
		osInfo.dwOSVersionInfoSize = uint32(unsafe.Sizeof(osInfo))

		ret, _, _ := rtlGetVersionFn.Call(uintptr(unsafe.Pointer(&osInfo)))

		if ret == 0 { // STATUS_SUCCESS
			info["build"] = int(osInfo.dwBuildNumber)
			info["versao_compilacao"] = fmt.Sprintf("%d.%d.%d",
				osInfo.dwMajorVersion,
				osInfo.dwMinorVersion,
				osInfo.dwBuildNumber)

			// Determinar o nome do SO com base na versão
			info["nome_so"] = getWindowsVersionName(osInfo.dwMajorVersion, osInfo.dwMinorVersion, osInfo.dwBuildNumber)
		}
	}

	// Obter nome do usuário atual
	if getUserNameFn != nil {
		var size uint32 = 260
		var buffer [260]uint16

		ret, _, _ := getUserNameFn.Call(
			uintptr(unsafe.Pointer(&buffer[0])),
			uintptr(unsafe.Pointer(&size)),
		)

		if ret != 0 {
			info["usuario_atual"] = syscall.UTF16ToString(buffer[:size-1]) // -1 para remover o terminador nulo
		}
	}

	// Obter informações adicionais do sistema
	info["arquitetura"] = getSystemArchitecture()

	// Obter usuário de execução (whoami)
	usuarioExecucao, err := executeCommand("whoami")
	if err == nil {
		info["usuario_execucao"] = strings.TrimSpace(usuarioExecucao)
	} else {
		info["usuario_execucao"] = "Desconhecido"
	}

	// Obter usuário atual (query user)
	usuarioAtual, err := executeCommand("query", "user")
	if err == nil {
		linhas := strings.Split(usuarioAtual, "\n")
		if len(linhas) > 1 { // Ignorar cabeçalho
			campos := strings.Fields(linhas[1])
			if len(campos) > 0 {
				// Garantir que o valor não seja sobrescrito apenas se estiver vazio
				if _, ok := info["usuario_atual"]; !ok {
					info["usuario_atual"] = campos[0]
				}
			}
		}
	}

	// Garantir que usuario_atual sempre tenha um valor
	if _, ok := info["usuario_atual"]; !ok {
		// Tentar obter do ambiente
		username := os.Getenv("USERNAME")
		if username != "" {
			info["usuario_atual"] = username
		} else {
			// Usar o mesmo valor de usuario_execucao como fallback
			if execUser, ok := info["usuario_execucao"]; ok {
				info["usuario_atual"] = execUser
			} else {
				info["usuario_atual"] = "Desconhecido"
			}
		}
	}

	// Obter informações sobre impressoras
	info["impressoras"] = getPrinterInfoNew()

	return info
}

// Função para obter o nome da versão do Windows
func getWindowsVersionName(major, minor, build uint32) string {
	// Determinar o nome base do Windows com base na versão
	var baseVersion string
	switch {
	case major == 10:
		if build >= 22000 {
			baseVersion = "Windows 11"
		} else {
			baseVersion = "Windows 10"
		}
	case major == 6 && minor == 3:
		baseVersion = "Windows 8.1"
	case major == 6 && minor == 2:
		baseVersion = "Windows 8"
	case major == 6 && minor == 1:
		baseVersion = "Windows 7"
	case major == 6 && minor == 0:
		baseVersion = "Windows Vista"
	case major == 5 && minor == 2:
		baseVersion = "Windows Server 2003"
	case major == 5 && minor == 1:
		baseVersion = "Windows XP"
	case major == 5 && minor == 0:
		baseVersion = "Windows 2000"
	default:
		return fmt.Sprintf("Windows (Versão %d.%d.%d)", major, minor, build)
	}

	// Obter a edição do Windows
	edition := getWindowsEdition()
	if edition != "" {
		return fmt.Sprintf("Microsoft %s %s", baseVersion, edition)
	}

	return fmt.Sprintf("Microsoft %s", baseVersion)
}

// Função para obter a edição do Windows através do registro
func getWindowsEdition() string {
	// Verificar se temos acesso ao registro
	if regOpenKeyExFn == nil || regQueryValueExFn == nil || regCloseKeyFn == nil {
		return ""
	}

	// Constantes para o registro do Windows
	const HKEY_LOCAL_MACHINE = 0x80000002
	const KEY_READ = 0x20019

	// Caminho para informações da edição do Windows
	keyPath, _ := syscall.UTF16PtrFromString("SOFTWARE\\Microsoft\\Windows NT\\CurrentVersion")

	var hKey syscall.Handle
	ret, _, _ := regOpenKeyExFn.Call(
		HKEY_LOCAL_MACHINE,
		uintptr(unsafe.Pointer(keyPath)),
		0,
		KEY_READ,
		uintptr(unsafe.Pointer(&hKey)),
	)

	if ret != 0 {
		return ""
	}
	defer regCloseKeyFn.Call(uintptr(hKey))

	// Primeiro, tentar obter o valor de EditionID
	edition := getRegistryString(hKey, "EditionID")
	if edition == "Desconhecido" {
		// Tentar obter o valor de ProductName como alternativa
		productName := getRegistryString(hKey, "ProductName")
		if productName != "Desconhecido" {
			// Extrair a edição do ProductName (geralmente contém o nome completo)
			parts := strings.Split(productName, " ")
			if len(parts) > 2 {
				// Retornar a parte após "Windows XX"
				return strings.Join(parts[2:], " ")
			}
		}
		return ""
	}

	// Mapear os códigos de edição para nomes mais amigáveis
	switch edition {
	case "Core":
		return "Home"
	case "CoreN":
		return "Home N"
	case "CoreSingleLanguage":
		return "Home Single Language"
	case "Professional":
		return "Pro"
	case "ProfessionalN":
		return "Pro N"
	case "Enterprise":
		return "Enterprise"
	case "EnterpriseN":
		return "Enterprise N"
	case "Education":
		return "Education"
	case "EducationN":
		return "Education N"
	case "IoTEnterprise":
		return "IoT Enterprise"
	case "ServerStandard":
		return "Server Standard"
	case "ServerDatacenter":
		return "Server Datacenter"
	default:
		return edition
	}
}

// Função auxiliar para obter strings do registro do Windows
// Esta função já existe em syscall_windows_dll.go, então podemos removê-la daqui
// e usar a versão centralizada
func getRegistryString(hKey syscall.Handle, valueName string) string {
	if regQueryValueExFn == nil {
		return "Desconhecido"
	}

	var bufSize uint32 = 128
	buf := make([]uint16, bufSize)
	valueNamePtr, _ := syscall.UTF16PtrFromString(valueName)

	ret, _, _ := regQueryValueExFn.Call(
		uintptr(hKey),
		uintptr(unsafe.Pointer(valueNamePtr)),
		0,
		0,
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&bufSize)),
	)

	if ret == 0 {
		return syscall.UTF16ToString(buf[:])
	}
	return "Desconhecido"
}

// Função para obter impressoras
// getPrinterInfoNew obtém informações sobre as impressoras instaladas no sistema
func getPrinterInfoNew() []map[string]interface{} {
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
				"Get-Printer | ForEach-Object { $name = $_.Name; $driver = $_.DriverName; $port = $_.PortName; $shared = $_.Shared; $shareName = $_.ShareName; $location = $_.Location; Write-Host \"$name|$driver|$port|$shared|$shareName|$location\" }")
		} else {
			// É uma única impressora
			cmd = exec.Command("powershell", "-Command",
				"$printer = Get-Printer; $name = $printer.Name; $driver = $printer.DriverName; $port = $printer.PortName; $shared = $printer.Shared; $shareName = $printer.ShareName; $location = $printer.Location; Write-Host \"$name|$driver|$port|$shared|$shareName|$location\"")
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
						sharedStr := strings.TrimSpace(parts[3])
						printer["compartilhada"] = (sharedStr == "True")
					} else {
						printer["compartilhada"] = false // Valor padrão
					}

					if len(parts) >= 5 {
						shareName := strings.TrimSpace(parts[4])
						if shareName != "" {
							printer["nome_compartilhamento"] = shareName
						}
					}

					if len(parts) >= 6 {
						location := strings.TrimSpace(parts[5])
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
			"Get-CimInstance -Class Win32_Printer | ForEach-Object { $name = $_.Name; $driver = $_.DriverName; $port = $_.PortName; $shared = $_.Shared; $shareName = $_.ShareName; $location = $_.Location; Write-Host \"$name|$driver|$port|$shared|$shareName|$location\" }")

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
						sharedStr := strings.TrimSpace(parts[3])
						printer["compartilhada"] = (sharedStr == "True")
					} else {
						printer["compartilhada"] = false // Valor padrão
					}

					if len(parts) >= 5 {
						shareName := strings.TrimSpace(parts[4])
						if shareName != "" {
							printer["nome_compartilhamento"] = shareName
						}
					}

					if len(parts) >= 6 {
						location := strings.TrimSpace(parts[5])
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
			cmd = exec.Command("wmic", "printer", "get", "name,drivername,portname,shared,sharename")
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
						nameEndIndex := len(fields) - 5
						if nameEndIndex < 1 {
							nameEndIndex = 1
						}

						name := strings.Join(fields[:nameEndIndex], " ")
						driver := fields[nameEndIndex+1]
						port := fields[nameEndIndex+2]
						sharedStr := fields[nameEndIndex+3]

						printer["nome"] = strings.TrimSpace(name)

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

								printer["driver"] = strings.TrimSpace(parts[startIndex+2])
								printer["porta"] = strings.TrimSpace(parts[startIndex+3])
								printer["compartilhada"] = (strings.ToLower(strings.TrimSpace(parts[startIndex+4])) == "true")

								if len(parts) > startIndex+5 {
									shareName := strings.TrimSpace(parts[startIndex+5])
									if shareName != "" && strings.ToLower(shareName) != "false" {
										printer["nome_compartilhamento"] = shareName
									}
								}
								printers = append(printers, printer)
							}
						}
					}
				}
			}
		}
	}

	// Garantir que todas as impressoras tenham os campos necessários
	for i := range printers {

		// Garantir que o campo compartilhada exista
		if _, ok := printers[i]["compartilhada"]; !ok {
			printers[i]["compartilhada"] = false
		}
	}

	return printers
}
