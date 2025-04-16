package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"time"
)

func main() {
	// Configurar flags de linha de comando
	agentIP := flag.String("agent", "", "IP do agente para atualizar (ex: 192.168.1.100:9999)")
	updateIP := flag.String("update-ip", "", "Novo IP do servidor de atualização (ex: http://10.0.0.1:9991)")
	flag.IntVar(&port, "port", 9991, "Porta do servidor HTTP")
	flag.DurationVar(&readTimeout, "read-timeout", 10*time.Second, "Timeout para leitura de requisições")
	flag.DurationVar(&writeTimeout, "write-timeout", 30*time.Second, "Timeout para escrita de respostas")
	flag.DurationVar(&idleTimeout, "idle-timeout", 120*time.Second, "Timeout para conexões ociosas")
	flag.IntVar(&maxHeaderMB, "max-header", 1, "Tamanho máximo do cabeçalho em MB")
	flag.Parse()

	// Carregar a chave privada
	var err error
	privateKey, err = loadPrivateKey("keys/private_key.pem")
	if err != nil {
		log.Printf("Aviso: Não foi possível carregar a chave privada: %v", err)
		log.Println("Algumas funcionalidades de segurança estarão indisponíveis")
	} else {
		log.Println("Chave privada carregada com sucesso")
	}

	// Verificar se é para atualizar um agente
	if *agentIP != "" && *updateIP != "" {
		if privateKey == nil {
			log.Fatalf("Erro: Chave privada necessária para atualizar agentes")
		}

		err := updateAgentServerIP(*agentIP, *updateIP)
		if err != nil {
			log.Fatalf("Erro ao atualizar agente: %v", err)
		}

		log.Printf("Agente %s atualizado com sucesso para usar o servidor %s", *agentIP, *updateIP)
		return
	}

	// Inicializar o mapa de clientes ativos
	activeClients = make(map[string]int)

	// Obter diretório atual
	currentDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Erro ao obter diretório atual: %v", err)
	}

	// Configurar o manipulador de arquivos estáticos
	fileServer := http.FileServer(http.Dir(currentDir))

	// Registrar handlers
	http.HandleFunc("/", fileServerHandler(fileServer))
	http.HandleFunc("/stats", statsHandler)
	http.HandleFunc("/api/update-server", handleUpdateServerIP)

	// Configurar o servidor HTTP com timeouts e limites
	server := &http.Server{
		Addr:           fmt.Sprintf(":%d", port),
		ReadTimeout:    readTimeout,
		WriteTimeout:   writeTimeout,
		IdleTimeout:    idleTimeout,
		MaxHeaderBytes: maxHeaderMB << 20, // Converter MB para bytes
	}

	// Exibir informações do servidor
	log.Printf("Servidor de atualizações iniciado em http://localhost:%d", port)
	log.Printf("Estatísticas disponíveis em http://localhost:%d/stats", port)
	log.Printf("Servindo arquivos do diretório: %s", currentDir)
	log.Printf("Sistema: %s %s", runtime.GOOS, runtime.GOARCH)

	// Verificar e exibir arquivos importantes
	checkImportantFiles(currentDir)

	// Iniciar rotina para exibir estatísticas periodicamente
	go showClientStats()

	// Iniciar o servidor
	log.Fatal(server.ListenAndServe())
}
