package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

// Variáveis globais
var (
	cachedSystemInfo    SystemInfo
	lastUpdateTime      time.Time
	lastUpdateCheckTime time.Time
	updateServerURL     string // Servidor de atualização - será carregado do banco de dados

	// Adicionar variáveis para controlar os intervalos
	systemInfoUpdateIntervalMinutes int
	updateCheckIntervalMinutes      int

	// Canais para sinalizar mudanças nos intervalos
	systemInfoIntervalChanged  chan bool
	updateCheckIntervalChanged chan bool
)

func main() {
	port := 9999

	// Verificar se a porta já está em uso (outra instância do agente já está rodando)
	if isPortInUse(port) {
		fmt.Printf("A porta %d já está em uso. Outra instância do agente já está em execução.\n", port)
		fmt.Println("Encerrando esta instância...")
		return
	}

	fmt.Println("Agente HTTP iniciado. Aguardando requisições...")

	// Inicializar canais para sinalizar mudanças nos intervalos
	systemInfoIntervalChanged = make(chan bool, 1)
	updateCheckIntervalChanged = make(chan bool, 1)

	// Verificar se estamos em um processo de atualização recente
	if isRecentlyUpdated() {
		fmt.Println("Atualização recente detectada, pulando verificação de atualizações inicial")
		// Criar um arquivo para marcar que a atualização foi concluída com sucesso
		markUpdateSuccess()
	}

	// Inicializa o banco de dados SQLite
	err := initDatabase()
	if err != nil {
		fmt.Printf("Erro ao inicializar banco de dados: %v\n", err)
		return
	}
	defer closeDatabase()

	// Verificar se há um arquivo version.txt na pasta do executável
	// e atualizar a versão no banco de dados se necessário
	updateVersionFromFile()

	// Carregar o IP do servidor de atualização do banco de dados
	serverIP, err := getUpdateServerIP()
	if err != nil {
		fmt.Printf("Erro ao obter IP do servidor de atualização: %v\n", err)
		fmt.Println("Usando IP de atualização padrão")
		updateServerURL = "http://10.46.102.245:9991"
	} else {
		updateServerURL = serverIP
		fmt.Printf("Usando servidor de atualização: %s\n", updateServerURL)
	}

	// Obter os intervalos de atualização configurados
	systemInfoUpdateIntervalMinutes, err = getSystemInfoUpdateInterval()
	if err != nil {
		fmt.Printf("Erro ao obter intervalo de atualização de informações: %v\n", err)
		fmt.Println("Usando intervalo padrão de 10 minutos")
		systemInfoUpdateIntervalMinutes = 10
	} else {
		fmt.Printf("Intervalo de atualização de informações: %d minutos\n", systemInfoUpdateIntervalMinutes)
	}

	updateCheckIntervalMinutes, err = getUpdateCheckInterval()
	if err != nil {
		fmt.Printf("Erro ao obter intervalo de verificação de atualizações: %v\n", err)
		fmt.Println("Usando intervalo padrão de 10 minutos")
		updateCheckIntervalMinutes = 10
	} else {
		fmt.Printf("Intervalo de verificação de atualizações: %d minutos\n", updateCheckIntervalMinutes)
	}

	// Verificar atualizações
	fmt.Println("Verificando atualizações disponíveis...")
	updateAvailable, latestVersion, err := checkForUpdates()
	if err != nil {
		fmt.Printf("Aviso: Não foi possível verificar atualizações: %v\n", err)
	} else if updateAvailable {
		fmt.Printf("Nova versão disponível: %s. Baixando atualização...\n", latestVersion)
		err = downloadAndUpdate(latestVersion)
		if err != nil {
			fmt.Printf("Erro ao baixar atualização: %v\n", err)
		} else {
			fmt.Println("Atualização baixada com sucesso. O aplicativo será reiniciado.")
			// Reiniciar o aplicativo
			restartApplication()
			return
		}
	} else {
		fmt.Println("O aplicativo está atualizado.")
	}

	// Criar tarefa agendada no Windows para inicialização automática
	if runtime.GOOS == "windows" {
		// Sempre criar a tarefa, independentemente de já existir ou não
		_, err := createStartupTask()
		if err != nil {
			fmt.Printf("Aviso: Não foi possível criar tarefa de inicialização: %v\n", err)
		} else {
			fmt.Println("Tarefa de inicialização automática criada/atualizada com sucesso.")
		}
	}

	// Verificar conectividade com a internet
	fmt.Println("Verificando conectividade com a internet...")
	internetAvailable := checkInternetConnectivity()
	if !internetAvailable {
		fmt.Println("Aviso: Não foi possível conectar à internet. O servidor continuará a inicialização.")
	} else {
		fmt.Println("Conectividade com a internet confirmada.")
	}

	// Limpar o banco de dados e coletar informações atualizadas
	fmt.Println("Limpando banco de dados e coletando informações atualizadas...")
	err = clearDatabase()
	if err != nil {
		fmt.Printf("Erro ao limpar banco de dados: %v\n", err)
	}

	// Coleta informações do sistema inicialmente
	cachedSystemInfo, err = collectSystemInfo()
	if err != nil {
		fmt.Printf("Erro ao coletar informações do sistema: %v\n", err)
		return
	}

	// Salva as informações coletadas no banco de dados
	err = saveSystemInfoToDB(cachedSystemInfo)
	if err != nil {
		fmt.Printf("Erro ao salvar informações no banco de dados: %v\n", err)
	}

	lastUpdateTime = time.Now()
	fmt.Println("Informações do sistema atualizadas e armazenadas em cache.")

	// Obter endereço IPv4 da máquina
	ipv4, err := getLocalIPv4()
	if err != nil {
		fmt.Printf("Aviso: Não foi possível obter o endereço IPv4: %v\n", err)
		fmt.Printf("Servidor rodando em http://localhost:%d\n", port)
	} else {
		fmt.Printf("Servidor rodando em http://%s:%d\n", ipv4, port)
	}

	// Verificando se o diretório de chaves existe
	exePath, err := os.Executable()
	if err != nil {
		fmt.Printf("Erro ao obter caminho do executável: %v\n", err)
		return
	}

	exeDir := filepath.Dir(exePath)
	keysDir := filepath.Join(exeDir, "keys")
	if _, err := os.Stat(keysDir); os.IsNotExist(err) {
		err = os.MkdirAll(keysDir, 0700)
		if err != nil {
			fmt.Printf("Erro ao criar diretório de chaves: %v\n", err)
			return
		}
		fmt.Printf("Diretório de chaves criado: %s\n", keysDir)
	}

	// Verificar e remover a chave privada se existir
	privateKeyPath := filepath.Join(keysDir, "private_key.pem")
	if _, err := os.Stat(privateKeyPath); err == nil {
		err = os.Remove(privateKeyPath)
		if err != nil {
			fmt.Printf("Aviso: Não foi possível remover a chave privada: %v\n", err)
		} else {
			fmt.Println("Chave privada removida por segurança")
		}
	}

	// Verificando se a chave pública existe
	publicKeyPath := filepath.Join(keysDir, "public_key.pem")
	if _, err := os.Stat(publicKeyPath); os.IsNotExist(err) {
		fmt.Printf("AVISO: Chave pública não encontrada em %s\n", publicKeyPath)
		fmt.Println("Por favor, gere as chaves usando o script generate_keys.go")
	} else {
		fmt.Printf("Chave pública carregada de: %s\n", publicKeyPath)
	}

	// Iniciar goroutines para gerenciar atualizações periódicas
	go manageSystemInfoUpdates()
	go manageUpdateChecks()

	// Inicializar o servidor HTTP
	initHTTPServer(port) // Change to use server package

	// Aguardar sinal para encerrar o programa
	waitForShutdown()
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

// markUpdateSuccess marca que uma atualização foi concluída com sucesso
func markUpdateSuccess() {
	exePath, err := os.Executable()
	if err != nil {
		return
	}

	exeDir := filepath.Dir(exePath)
	successFlagPath := filepath.Join(exeDir, ".update_success")

	// Criar arquivo de marcação
	os.WriteFile(successFlagPath, []byte(time.Now().String()), 0644)
}

// manageSystemInfoUpdates gerencia as atualizações periódicas das informações do sistema
func manageSystemInfoUpdates() {
	// Definir o ticker inicial
	ticker := time.NewTicker(time.Duration(systemInfoUpdateIntervalMinutes) * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Atualizar apenas informações dinâmicas
			updateDynamicInfo(&cachedSystemInfo)

			// Salvar as informações atualizadas no banco de dados
			err := saveSystemInfoToDB(cachedSystemInfo)
			if err != nil {
				fmt.Printf("Erro ao salvar informações atualizadas: %v\n", err)
			}

			lastUpdateTime = time.Now()

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
			fmt.Printf("Verificando atualizações disponíveis (intervalo: %d minutos)...\n", updateCheckIntervalMinutes)
			updateAvailable, latestVersion, err := checkForUpdates()
			if err != nil {
				fmt.Printf("Aviso: Não foi possível verificar atualizações: %v\n", err)
			} else if updateAvailable {
				fmt.Printf("Nova versão disponível: %s. Baixando atualização...\n", latestVersion)
				err = downloadAndUpdate(latestVersion)
				if err != nil {
					fmt.Printf("Erro ao baixar atualização: %v\n", err)
				} else {
					fmt.Println("Atualização baixada com sucesso. O aplicativo será reiniciado.")
					// Reiniciar o aplicativo
					restartApplication()
					return
				}
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
