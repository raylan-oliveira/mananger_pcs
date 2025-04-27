package main

import (
	"fmt"
	"os"
	"path/filepath"
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

	// Inicializar o banco de dados SQLite
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
		// Definir o valor padrão
		updateServerURL = "http://10.46.102.245:9991"

		// Tentar salvar o valor padrão no banco de dados para uso futuro
		err = updateServerIP(updateServerURL)
		if err != nil {
			fmt.Printf("Erro ao salvar IP padrão no banco de dados: %v\n", err)
		} else {
			fmt.Println("IP padrão salvo no banco de dados com sucesso")
		}
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
	fmt.Println("[main] Verificando atualizações disponíveis...")
	updateAvailable, latestVersion, err := checkForUpdates()
	if err != nil {
		fmt.Printf("Aviso: Não foi possível verificar atualizações: %v\n", err)
	} else if updateAvailable {
		fmt.Printf("Nova versão disponível: %s. Baixando atualização...\n", latestVersion)
		err = downloadAndUpdate(latestVersion, true) // Passar true para indicar que é verificação inicial
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
	// Sempre criar a tarefa, independentemente de já existir ou não
	_, err = createStartupTask()
	if err != nil {
		fmt.Printf("Aviso: Não foi possível criar tarefa de inicialização: %v\n", err)
	} else {
		fmt.Println("Tarefa de inicialização automática criada/atualizada com sucesso.")
	}

	// // Verificar conectividade com a internet
	// fmt.Println("Verificando conectividade com a internet...")
	// internetAvailable := checkInternetConnectivity()
	// if !internetAvailable {
	// 	fmt.Println("Aviso: Não foi possível conectar à internet. O servidor continuará a inicialização.")
	// } else {
	// 	fmt.Println("Conectividade com a internet confirmada.")
	// }

	// Limpar o banco de dados e coletar informações atualizadas
	fmt.Println("Limpando banco de dados e coletando informações atualizadas...")
	err = clearDatabase()
	if err != nil {
		fmt.Printf("Erro ao limpar banco de dados: %v\n", err)
	}

	// Coleta informações do sistema inicialmente
	cachedSystemInfo, err = collectAllInfo()
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
