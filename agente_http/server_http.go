package main

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// Variáveis globais para controle do servidor HTTP
var (
	httpServer     *http.Server
	serverShutdown chan bool
)

// Inicializa o servidor HTTP
func initHTTPServer(port int) {
	// Configurando o servidor HTTP
	mux := http.NewServeMux()
	mux.HandleFunc("/", systemInfoHandler)
	mux.HandleFunc("/update-server", updateServerIPHandler)
	mux.HandleFunc("/update-system-info-interval", updateSystemInfoIntervalHandler)
	mux.HandleFunc("/update-check-interval", updateCheckIntervalHandler)

	// Novos endpoints para componentes individuais
	mux.HandleFunc("/cpu", cpuHandler)
	mux.HandleFunc("/discos", discosHandler)
	mux.HandleFunc("/gpu", gpuHandler)
	mux.HandleFunc("/hardware", hardwareHandler)
	mux.HandleFunc("/memoria", memoriaHandler)
	mux.HandleFunc("/processos", processosHandler)
	mux.HandleFunc("/rede", redeHandler)
	mux.HandleFunc("/sistema", sistemaHandler)
	mux.HandleFunc("/agente", agenteHandler)

	// Criar o servidor com configurações personalizadas
	httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	// Canal para sinalizar encerramento
	serverShutdown = make(chan bool)

	// Iniciar o servidor em uma goroutine
	go func() {
		fmt.Printf("Iniciando servidor HTTP na porta %d em todas as interfaces de rede...\n", port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Erro ao iniciar servidor HTTP: %v\n", err)
		}

		// Sinalizar que o servidor foi encerrado
		serverShutdown <- true
	}()
}

// Encerra o servidor HTTP graciosamente
func shutdownHTTPServer() {
	if httpServer != nil {
		fmt.Println("Encerrando servidor HTTP...")

		// Criar contexto com timeout para encerramento gracioso
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Tentar encerrar o servidor graciosamente
		if err := httpServer.Shutdown(ctx); err != nil {
			fmt.Printf("Erro ao encerrar servidor HTTP: %v\n", err)
		}

		// Aguardar sinal de encerramento
		<-serverShutdown
		fmt.Println("Servidor HTTP encerrado com sucesso")
	}
}
