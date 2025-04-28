package main

import (
	"fmt"
	"syscall"
	"time"
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

// Inicializa todas as DLLs e procedimentos necessários
func initGetSystemInfo() error {
	// Se já inicializou, retorna imediatamente
	if dllsInitialized {
		return nil
	}

	var err error

	// Carregar kernel32.dll
	if kernel32DLL == nil {
		kernel32DLL, err = syscall.LoadDLL("kernel32.dll")
		if err != nil {
			return err
		}

		// Carregar procedimentos do kernel32.dll
		getSystemInfoFn, err = kernel32DLL.FindProc("GetSystemInfo")
		if err != nil {
			return err
		}

		getComputerNameExFn, _ = kernel32DLL.FindProc("GetComputerNameExW")
		getTickCount64Fn, _ = kernel32DLL.FindProc("GetTickCount64")
		getSystemTimeFn, _ = kernel32DLL.FindProc("GetSystemTime")
		getUserNameFn, _ = kernel32DLL.FindProc("GetUserNameW")
		isWow64ProcessFn, _ = kernel32DLL.FindProc("IsWow64Process")
		getCurrentProcessFn, _ = kernel32DLL.FindProc("GetCurrentProcess")
	}

	// Carregar ntdll.dll
	if ntdllDLL == nil {
		ntdllDLL, err = syscall.LoadDLL("ntdll.dll")
		if err == nil {
			rtlGetVersionFn, _ = ntdllDLL.FindProc("RtlGetVersion")
		}
	}

	// Carregar advapi32.dll
	if advapi32DLL == nil {
		advapi32DLL, err = syscall.LoadDLL("advapi32.dll")
		if err == nil {
			regOpenKeyExFn, _ = advapi32DLL.FindProc("RegOpenKeyExW")
			regQueryValueExFn, _ = advapi32DLL.FindProc("RegQueryValueExW")
			regCloseKeyFn, _ = advapi32DLL.FindProc("RegCloseKey")
			regEnumKeyExFn, _ = advapi32DLL.FindProc("RegEnumKeyExW") // Add this line
		}
	}

	// Marcar como inicializado
	dllsInitialized = true

	return nil
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

	// Obter informações do sistema
	sysInfo, err := getSystemInfoData()
	if err != nil {
		info["erro"] = err.Error()
		return info
	}

	// Preencher o mapa de informações
	info["tipo_processador"] = int(sysInfo.dwProcessorType)
	info["nivel_processador"] = int(sysInfo.wProcessorLevel)
	info["revisao_processador"] = int(sysInfo.wProcessorRevision)
	info["num_processadores"] = int(sysInfo.dwNumberOfProcessors)

	// Determinar a arquitetura do processador
	switch sysInfo.dwOemID >> 8 {
	case 0:
		info["arquitetura"] = "x86"
	case 9:
		info["arquitetura"] = "x64"
	case 5:
		info["arquitetura"] = "ARM"
	case 12:
		info["arquitetura"] = "ARM64"
	default:
		info["arquitetura"] = fmt.Sprintf("Desconhecida (%d)", sysInfo.dwOemID>>8)
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
		}
	}

	// Adicionar informações básicas se não foram obtidas
	if _, ok := info["fabricante"]; !ok {
		info["fabricante"] = "Desconhecido"
	}
	if _, ok := info["modelo"]; !ok {
		info["modelo"] = "Desconhecido"
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

	// Verificar se temos os procedimentos necessários
	if isWow64ProcessFn == nil || getCurrentProcessFn == nil {
		return "Desconhecido"
	}

	// Obter o handle do processo atual
	handle, _, _ := getCurrentProcessFn.Call()

	var isWow64 bool
	ret, _, _ := isWow64ProcessFn.Call(handle, uintptr(unsafe.Pointer(&isWow64)))

	if ret != 0 {
		if isWow64 {
			return "x64" // Processo 32 bits rodando em sistema 64 bits
		}

		// Verificar se é um sistema 64 bits nativo
		sysInfo, err := getSystemInfoData()
		if err == nil {
			// Verificar a arquitetura do processador
			switch sysInfo.wProcessorLevel {
			case 6:
				return "x64"
			default:
				return "x86"
			}
		}
	}

	return "x86" // Padrão para sistemas 32 bits
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
			info["versao_major"] = int(osInfo.dwMajorVersion)
			info["versao_minor"] = int(osInfo.dwMinorVersion)
			info["build"] = int(osInfo.dwBuildNumber)
			info["versao_compilacao"] = fmt.Sprintf("%d.%d.%d",
				osInfo.dwMajorVersion,
				osInfo.dwMinorVersion,
				osInfo.dwBuildNumber)

			// Determinar o nome do SO com base na versão
			info["nome_so"] = getWindowsVersionName(osInfo.dwMajorVersion, osInfo.dwMinorVersion, osInfo.dwBuildNumber)
		}
	}

	// Obter informações de uptime
	if getTickCount64Fn != nil && getSystemTimeFn != nil {
		uptimeMs, _, _ := getTickCount64Fn.Call()
		uptimeMinutes := uptimeMs / (1000 * 60) // Converter de milissegundos para minutos
		info["uptime"] = int64(uptimeMinutes)

		// Calcular o horário do último boot
		var systemTime systemTime
		getSystemTimeFn.Call(uintptr(unsafe.Pointer(&systemTime)))

		now := time.Date(
			int(systemTime.wYear),
			time.Month(systemTime.wMonth),
			int(systemTime.wDay),
			int(systemTime.wHour),
			int(systemTime.wMinute),
			int(systemTime.wSecond),
			int(systemTime.wMilliseconds)*1000000,
			time.UTC,
		)

		bootTime := now.Add(-time.Duration(uptimeMs) * time.Millisecond)
		info["ultimo_boot"] = bootTime.Format("02/01/2006 15:04:05")
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

	info["arquitetura"] = getSystemArchitecture()

	return info
}

// Função para obter o nome da versão do Windows
func getWindowsVersionName(major, minor, build uint32) string {
	// Determinar o nome do Windows com base na versão
	switch {
	case major == 10:
		if build >= 22000 {
			return "Windows 11"
		}
		return "Windows 10"
	case major == 6 && minor == 3:
		return "Windows 8.1"
	case major == 6 && minor == 2:
		return "Windows 8"
	case major == 6 && minor == 1:
		return "Windows 7"
	case major == 6 && minor == 0:
		return "Windows Vista"
	case major == 5 && minor == 2:
		return "Windows Server 2003"
	case major == 5 && minor == 1:
		return "Windows XP"
	case major == 5 && minor == 0:
		return "Windows 2000"
	default:
		return fmt.Sprintf("Windows (Versão %d.%d.%d)", major, minor, build)
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
