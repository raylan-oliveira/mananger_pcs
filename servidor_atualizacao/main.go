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

// Configurações do servidor
var (
	port         int
	readTimeout  time.Duration
	writeTimeout time.Duration
	idleTimeout  time.Duration
	maxHeaderMB  int
)

func main() {
	// Configurar flags de linha de comando
	flag.IntVar(&port, "port", 9991, "Porta do servidor HTTP")
	flag.DurationVar(&readTimeout, "read-timeout", 10*time.Second, "Timeout para leitura de requisições")
	flag.DurationVar(&writeTimeout, "write-timeout", 30*time.Second, "Timeout para escrita de respostas")
	flag.DurationVar(&idleTimeout, "idle-timeout", 120*time.Second, "Timeout para conexões ociosas")
	flag.IntVar(&maxHeaderMB, "max-header", 1, "Tamanho máximo do cabeçalho em MB")
	flag.Parse()

	// Obter diretório atual
	currentDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Erro ao obter diretório atual: %v", err)
	}

	// Configurar o manipulador de arquivos estáticos
	fileServer := http.FileServer(http.Dir(currentDir))

	// Registrar handlers
	http.HandleFunc("/", fileServerHandler(fileServer))

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

	log.Printf("Servindo arquivos do diretório: %s", currentDir)
	log.Printf("Sistema: %s %s", runtime.GOOS, runtime.GOARCH)

	// Verificar e exibir arquivos importantes
	checkImportantFiles(currentDir)

	// Iniciar rotina para limpar recursos periodicamente
	go cleanupResources()

	// Iniciar o servidor
	log.Fatal(server.ListenAndServe())
}

// cleanupResources limpa recursos periodicamente para evitar vazamentos de memória
func cleanupResources() {
	ticker := time.NewTicker(6 * time.Hour) // Aumentado de 1 para 6 horas
	defer ticker.Stop()

	for range ticker.C {
		// Registrar uso de memória atual
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		// Forçar coleta de lixo apenas se o uso de memória estiver alto
		if float64(m.Alloc)/1024/1024 > 100 { // Mais de 100MB alocados
			runtime.GC()
			log.Printf("Coleta de lixo forçada - Memória antes: %.2f MB", float64(m.Alloc)/1024/1024)

			// Atualizar estatísticas após GC
			runtime.ReadMemStats(&m)
		}

		log.Printf("Estatísticas de memória - Alocada: %.2f MB, Sistema: %.2f MB",
			float64(m.Alloc)/1024/1024,
			float64(m.Sys)/1024/1024)
	}
}
