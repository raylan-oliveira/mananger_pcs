package main

import (
	"syscall"
	"unsafe"
)

func getDiskInfoSyscall() map[string]interface{} {
	info := make(map[string]interface{})
	disks := make([]map[string]interface{}, 0)

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

	// Obter as letras de unidade disponíveis
	drives, _, _ := getLogicalDrivesFn.Call()
	
	// Iterar sobre as letras de unidade (A-Z)
	for i := 0; i < 26; i++ {
		if drives&(1<<i) == 0 {
			continue
		}
		
		driveLetter := string('A' + i)
		rootPath := driveLetter + ":\\"
		rootPathPtr, _ := syscall.UTF16PtrFromString(rootPath)
		
		// Obter informações de espaço em disco
		var freeBytesAvailable, totalBytes, totalFreeBytes uint64
		ret, _, _ := getDiskFreeSpaceExFn.Call(
			uintptr(unsafe.Pointer(rootPathPtr)),
			uintptr(unsafe.Pointer(&freeBytesAvailable)),
			uintptr(unsafe.Pointer(&totalBytes)),
			uintptr(unsafe.Pointer(&totalFreeBytes)),
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
		
		// Adicionar informações do disco
		disk := make(map[string]interface{})
		disk["letra"] = driveLetter
		disk["rotulo"] = volumeName
		disk["sistema_arquivos"] = fileSystemName
		disk["tamanho_total"] = totalBytes
		disk["espaco_livre"] = totalFreeBytes
		disk["porcentagem_livre"] = float64(totalFreeBytes) / float64(totalBytes) * 100
		
		disks = append(disks, disk)
	}
	
	info["discos"] = disks
	
	return info
}
