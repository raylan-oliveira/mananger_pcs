package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Variável global para armazenar as informações do systeminfo
var systemInfoCache map[string]interface{}
var systemInfoCacheTime time.Time
var systemInfoCacheMutex sync.Mutex

// parseUint64 converte uma string para uint64, retornando 0 em caso de erro
func parseUint64(s string) uint64 {
	// Tentar converter diretamente
	val, err := strconv.ParseUint(s, 10, 64)
	if err == nil {
		return val
	}

	// Tentar remover caracteres não numéricos e converter novamente
	var numStr string
	for _, c := range s {
		if c >= '0' && c <= '9' {
			numStr += string(c)
		}
	}

	if numStr == "" {
		return 0
	}

	val, err = strconv.ParseUint(numStr, 10, 64)
	if err != nil {
		return 0
	}

	return val
}

// isPortInUse verifica se a porta especificada já está em uso
func isPortInUse(port int) bool {
	// Tenta fazer um bind na porta para verificar se está disponível
	address := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", address)

	// Se não conseguir fazer o bind, a porta está em uso
	if err != nil {
		return true
	}

	// Se conseguiu fazer o bind, a porta está livre
	// Fecha o listener para liberar a porta
	listener.Close()
	return false
}

// Função para aguardar sinais de encerramento
func waitForShutdown() {
	// Criar canal para sinais do sistema operacional
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// Aguardar sinal
	<-sigs

	// Encerrar servidor HTTP
	shutdownHTTPServer() // Change to use server package

	fmt.Println("Programa encerrado")
}

// updateVersionFromFile verifica se existe um arquivo version.txt
// e atualiza a versão no banco de dados se necessário
func updateVersionFromFile() {
	exePath, err := os.Executable()
	if err != nil {
		fmt.Printf("Erro ao obter caminho do executável: %v\n", err)
		return
	}

	exeDir := filepath.Dir(exePath)
	versionPath := filepath.Join(exeDir, "version.txt")

	// Verificar se o arquivo existe
	if _, err := os.Stat(versionPath); os.IsNotExist(err) {
		return
	}

	// Ler o conteúdo do arquivo
	content, err := os.ReadFile(versionPath)
	if err != nil {
		fmt.Printf("Erro ao ler arquivo de versão: %v\n", err)
		return
	}

	// Obter a versão do arquivo
	fileVersion := strings.TrimSpace(string(content))

	// Verificar se a versão está no formato válido
	if !isValidVersionFormat(fileVersion) {
		fmt.Printf("Formato de versão inválido no arquivo: %s\n", fileVersion)
		return
	}

	// Obter a versão atual do banco de dados
	currentVersion, err := getCurrentVersion()
	if err != nil {
		fmt.Printf("Erro ao obter versão atual do banco de dados: %v\n", err)
		return
	}

	// Atualizar a versão no banco de dados se for diferente
	if fileVersion != currentVersion {
		fmt.Printf("Atualizando versão no banco de dados: %s -> %s\n", currentVersion, fileVersion)
		err = updateVersion(fileVersion)
		if err != nil {
			fmt.Printf("Erro ao atualizar versão no banco de dados: %v\n", err)
		}
	}
}

// manageSystemInfoUpdates gerencia as atualizações periódicas das informações do sistema
func manageSystemInfoUpdates() {
	// Definir o ticker inicial
	ticker := time.NewTicker(time.Duration(systemInfoUpdateIntervalMinutes) * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Coletar informações atualizadas do sistema
			infoAtualizada, err := collectAllInfoSyscall()
			if err != nil {
				fmt.Printf("Erro ao coletar informações atualizadas: %v\n", err)
				continue
			}

			// Salvar as informações atualizadas no banco de dados
			err = saveSystemInfoToDB(infoAtualizada)
			if err != nil {
				fmt.Printf("Erro ao salvar informações atualizadas: %v\n", err)
			}

			lastUpdateTime = time.Now()
			fmt.Println("Informações do sistema atualizadas e salvas no banco de dados.")

		case <-systemInfoIntervalChanged:
			// O intervalo foi alterado, recriar o ticker
			ticker.Stop()
			ticker = time.NewTicker(time.Duration(systemInfoUpdateIntervalMinutes) * time.Minute)
			fmt.Printf("Intervalo de atualização de informações alterado para: %d minutos\n", systemInfoUpdateIntervalMinutes)
		}
	}
}

// manageUpdateChecks gerencia as verificações periódicas de atualizações
func manageUpdateChecks() {
	// Definir o ticker inicial
	ticker := time.NewTicker(time.Duration(updateCheckIntervalMinutes) * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// É hora de verificar atualizações
			fmt.Printf("[manageUpdateChecks] Verificando atualizações disponíveis (intervalo: %d minutos)...\n", updateCheckIntervalMinutes)
			updateAvailable, latestVersion, err := checkForUpdates()
			if err != nil {
				fmt.Printf("Aviso: Não foi possível verificar atualizações: %v\n", err)
			} else if updateAvailable {
				fmt.Printf("Nova versão disponível: %s. Baixando atualização...\n", latestVersion)
				// Executar o download e atualização em uma goroutine separada
				go func(version string) {
					err = downloadAndUpdate(version, false) // Passar false para verificações periódicas
					if err != nil {
						fmt.Printf("Erro ao baixar atualização: %v\n", err)
					} else {
						fmt.Println("Atualização baixada com sucesso. O aplicativo será reiniciado.")
						// Reiniciar o aplicativo
						restartApplication()
					}
				}(latestVersion)

				// Continuar processando normalmente
				fmt.Println("Iniciando download da atualização em segundo plano...")
			} else {
				fmt.Println("O aplicativo está atualizado.")
			}

			lastUpdateCheckTime = time.Now()

		case <-updateCheckIntervalChanged:
			// O intervalo foi alterado, recriar o ticker
			ticker.Stop()
			ticker = time.NewTicker(time.Duration(updateCheckIntervalMinutes) * time.Minute)
			fmt.Printf("Intervalo de verificação de atualizações alterado para: %d minutos\n", updateCheckIntervalMinutes)
		}
	}
}
