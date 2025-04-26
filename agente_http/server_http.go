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
	mux            *http.ServeMux // Adicionar esta linha para definir o multiplexer
)

// Inicializa o servidor HTTP
func initHTTPServer(port int) {
	// Inicializar o multiplexer
	mux = http.NewServeMux() // Adicionar esta linha para inicializar o multiplexer

	// Middleware para adicionar cabeçalhos CORS
	corsMiddleware := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next(w, r)
		}
	}

	// Registrar handlers com middleware CORS
	mux.HandleFunc("/", corsMiddleware(quickSystemInfoHandlerDataBase))
	mux.HandleFunc("/info-all", corsMiddleware(systemInfoHandler)) // Novo endpoint para consultas rápidas
	mux.HandleFunc("/update-server", corsMiddleware(updateServerIPHandler))
	mux.HandleFunc("/update-system-info-interval", corsMiddleware(updateSystemInfoIntervalHandler))
	mux.HandleFunc("/update-check-interval", corsMiddleware(updateCheckIntervalHandler))
	mux.HandleFunc("/cpu", corsMiddleware(cpuHandler))
	mux.HandleFunc("/discos", corsMiddleware(discosHandler))
	mux.HandleFunc("/gpu", corsMiddleware(gpuHandler))
	mux.HandleFunc("/hardware", corsMiddleware(hardwareHandler))
	mux.HandleFunc("/memoria", corsMiddleware(memoriaHandler))
	mux.HandleFunc("/processos", corsMiddleware(processosHandler))
	mux.HandleFunc("/rede", corsMiddleware(redeHandler))
	mux.HandleFunc("/sistema", corsMiddleware(sistemaHandler))
	mux.HandleFunc("/agente", corsMiddleware(agenteHandler))
	mux.HandleFunc("/execute-command", corsMiddleware(commandHandler)) // Novo endpoint para execução de comandos

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
