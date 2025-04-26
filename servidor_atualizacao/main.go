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

	// Configurar o servidor HTTP com timeouts e limites
	server := &http.Server{
		Addr:           fmt.Sprintf(":%d", port),
		ReadTimeout:    readTimeout,
		WriteTimeout:   writeTimeout,
		IdleTimeout:    idleTimeout,
		MaxHeaderBytes: maxHeaderMB << 20, // Converter MB para bytes
	}

	// Obter endereço IPv4 da máquina
	ipv4, err := getLocalIPv4()
	if err != nil {
		log.Printf("Aviso: Não foi possível obter o endereço IPv4: %v", err)
		log.Printf("Servidor de atualizações iniciado em http://localhost:%d", port)
	} else {
		log.Printf("Servidor de atualizações iniciado em http://%s:%d", ipv4, port)
	}
	
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
