package main

import (
	"syscall"
	"unsafe"
)

func getGPUInfoSyscall() map[string]interface{} {
	info := make(map[string]interface{})
	gpus := make([]map[string]interface{}, 0)

	// Inicializar as DLLs e procedimentos
	err := initWindowsDLLs()
	if err != nil {
		info["erro"] = err.Error()
		return info
	}

	// Verificar se temos os procedimentos necessários
	if regOpenKeyExFn == nil || regQueryValueExFn == nil || regCloseKeyFn == nil || regEnumKeyExFn == nil {
		info["erro"] = "Não foi possível carregar funções do registro"
		return info
	}

	// Constantes para o registro do Windows
	const HKEY_LOCAL_MACHINE = 0x80000002
	const KEY_READ = 0x20019

	// Caminho para informações de display
	keyPath, _ := syscall.UTF16PtrFromString("SYSTEM\\CurrentControlSet\\Control\\Class\\{4d36e968-e325-11ce-bfc1-08002be10318}")

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

		// Enumerar as subchaves (cada uma representa um adaptador de vídeo)
		var i uint32 = 0
		nameBuffer := make([]uint16, 256)

		for {
			var nameSize uint32 = 256
			var lastWriteTime syscall.Filetime

			ret, _, _ := regEnumKeyExFn.Call(
				uintptr(hKey),
				uintptr(i),
				uintptr(unsafe.Pointer(&nameBuffer[0])),
				uintptr(unsafe.Pointer(&nameSize)),
				0,
				0,
				0,
				uintptr(unsafe.Pointer(&lastWriteTime)),
			)

			if ret != 0 {
				break // Não há mais subchaves
			}

			subKeyName := syscall.UTF16ToString(nameBuffer[:nameSize])

			// Ignorar chaves que não são numéricas (0000, 0001, etc.)
			if len(subKeyName) != 4 {
				i++
				continue
			}

			// Abrir a subchave
			subKeyPathStr := "SYSTEM\\CurrentControlSet\\Control\\Class\\{4d36e968-e325-11ce-bfc1-08002be10318}\\" + subKeyName
			subKeyPath, _ := syscall.UTF16PtrFromString(subKeyPathStr)
			var hSubKey syscall.Handle

			ret, _, _ = regOpenKeyExFn.Call(
				HKEY_LOCAL_MACHINE,
				uintptr(unsafe.Pointer(subKeyPath)),
				0,
				KEY_READ,
				uintptr(unsafe.Pointer(&hSubKey)),
			)

			if ret == 0 {
				gpu := make(map[string]interface{})

				// Obter descrição do dispositivo
				gpu["nome"] = getRegistryString(hSubKey, "DriverDesc")

				// Obter versão do driver
				gpu["driver_versao"] = getRegistryString(hSubKey, "DriverVersion")

				// Obter data do driver
				gpu["driver_data"] = getRegistryString(hSubKey, "DriverDate")

				// Adicionar à lista se tiver um nome válido
				if gpu["nome"] != "Desconhecido" && gpu["nome"] != "" {
					gpus = append(gpus, gpu)
				}

				regCloseKeyFn.Call(uintptr(hSubKey))
			}

			i++
		}
	}

	// Se não encontrou nenhuma GPU, adicionar uma entrada genérica
	if len(gpus) == 0 {
		gpu := make(map[string]interface{})
		gpu["nome"] = "Adaptador de Vídeo Desconhecido"
		gpu["driver_versao"] = "Desconhecida"
		gpus = append(gpus, gpu)
	}

	info["gpus"] = gpus

	return info
}
