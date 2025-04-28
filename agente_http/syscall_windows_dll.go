package main

import (
	"syscall"
)

// Variáveis globais para armazenar as DLLs e procedimentos
var (
	// DLLs
	kernel32DLL *syscall.DLL
	ntdllDLL    *syscall.DLL
	advapi32DLL *syscall.DLL
	psapiDLL    *syscall.DLL
	iphlpapiDLL *syscall.DLL

	// Procedimentos do kernel32.dll
	getSystemInfoFn            *syscall.Proc
	getComputerNameExFn        *syscall.Proc
	getTickCount64Fn           *syscall.Proc
	getSystemTimeFn            *syscall.Proc
	getUserNameFn              *syscall.Proc
	isWow64ProcessFn           *syscall.Proc
	getCurrentProcessFn        *syscall.Proc
	getLogicalDrivesFn         *syscall.Proc
	getDiskFreeSpaceExFn       *syscall.Proc
	getVolumeInformationFn     *syscall.Proc
	globalMemoryStatusExFn     *syscall.Proc
	createToolhelp32SnapshotFn *syscall.Proc
	process32FirstFn           *syscall.Proc
	process32NextFn            *syscall.Proc
	openProcessFn              *syscall.Proc
	closeHandleFn              *syscall.Proc

	// Procedimentos do ntdll.dll
	rtlGetVersionFn *syscall.Proc

	// Procedimentos do advapi32.dll
	regOpenKeyExFn    *syscall.Proc
	regQueryValueExFn *syscall.Proc
	regCloseKeyFn     *syscall.Proc
	regEnumKeyExFn    *syscall.Proc

	// Procedimentos do psapi.dll
	getProcessMemoryInfoFn *syscall.Proc

	// Procedimentos do iphlpapi.dll
	getNetworkParamsFn *syscall.Proc

	// Flag para indicar se a inicialização foi concluída
	dllsInitialized bool
)

// Constantes para GetComputerNameExW
const (
	ComputerNameNetBIOS                   = 0
	ComputerNameDnsHostname               = 1
	ComputerNameDnsDomain                 = 2
	ComputerNameDnsFullyQualified         = 3
	ComputerNamePhysicalNetBIOS           = 4
	ComputerNamePhysicalDnsHostname       = 5
	ComputerNamePhysicalDnsDomain         = 6
	ComputerNamePhysicalDnsFullyQualified = 7
	ComputerNameMax                       = 8
)

// Inicializa todas as DLLs e procedimentos necessários
func initWindowsDLLs() error {
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
		getSystemInfoFn, _ = kernel32DLL.FindProc("GetSystemInfo")
		getComputerNameExFn, _ = kernel32DLL.FindProc("GetComputerNameExW")
		getTickCount64Fn, _ = kernel32DLL.FindProc("GetTickCount64")
		getSystemTimeFn, _ = kernel32DLL.FindProc("GetSystemTime")
		getUserNameFn, _ = kernel32DLL.FindProc("GetUserNameW")
		isWow64ProcessFn, _ = kernel32DLL.FindProc("IsWow64Process")
		getCurrentProcessFn, _ = kernel32DLL.FindProc("GetCurrentProcess")
		getLogicalDrivesFn, _ = kernel32DLL.FindProc("GetLogicalDrives")
		getDiskFreeSpaceExFn, _ = kernel32DLL.FindProc("GetDiskFreeSpaceExW")
		getVolumeInformationFn, _ = kernel32DLL.FindProc("GetVolumeInformationW")
		globalMemoryStatusExFn, _ = kernel32DLL.FindProc("GlobalMemoryStatusEx")
		createToolhelp32SnapshotFn, _ = kernel32DLL.FindProc("CreateToolhelp32Snapshot")
		process32FirstFn, _ = kernel32DLL.FindProc("Process32FirstW")
		process32NextFn, _ = kernel32DLL.FindProc("Process32NextW")
		openProcessFn, _ = kernel32DLL.FindProc("OpenProcess")
		closeHandleFn, _ = kernel32DLL.FindProc("CloseHandle")
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
			regEnumKeyExFn, _ = advapi32DLL.FindProc("RegEnumKeyExW")
		}
	}

	// Carregar psapi.dll
	if psapiDLL == nil {
		psapiDLL, err = syscall.LoadDLL("psapi.dll")
		if err == nil {
			getProcessMemoryInfoFn, _ = psapiDLL.FindProc("GetProcessMemoryInfo")
		}
	}

	// Carregar iphlpapi.dll
	if iphlpapiDLL == nil {
		iphlpapiDLL, err = syscall.LoadDLL("iphlpapi.dll")
		if err == nil {
			getNetworkParamsFn, _ = iphlpapiDLL.FindProc("GetNetworkParams")
		}
	}

	// Marcar como inicializado
	dllsInitialized = true

	return nil
}
