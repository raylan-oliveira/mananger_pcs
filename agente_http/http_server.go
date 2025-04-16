package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Constante para controlar se os dados devem ser criptografados
const encriptado = true

// Handler para fornecer informações do sistema
func systemInfoHandler(w http.ResponseWriter, r *http.Request) {
	// Verificar se é hora de atualizar o cache (a cada 5 minutos)
	if time.Since(lastUpdateTime).Minutes() >= 5 {
		fmt.Println("Atualizando informações do sistema...")

		// Atualizar apenas informações dinâmicas
		updateDynamicInfo(&cachedSystemInfo)

		// Salvar as informações atualizadas no banco de dados
		err := saveSystemInfoToDB(cachedSystemInfo)
		if err != nil {
			fmt.Printf("Erro ao salvar informações atualizadas: %v\n", err)
		}

		// Verificar atualizações do agente após atualizar informações do sistema
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

		lastUpdateTime = time.Now()
	}

	// Obter a versão atual do agente do banco de dados
	versaoAgente, err := getCurrentVersion()
	if err != nil {
		fmt.Printf("Erro ao obter versão do agente: %v\n", err)
		versaoAgente = "desconhecida"
	}

	// Criar uma cópia da estrutura para não modificar o cache
	infoComVersao := cachedSystemInfo

	// Adicionar a versão do agente à estrutura
	infoComVersao.VersaoAgente = versaoAgente

	// Converter para JSON
	jsonData, err := json.MarshalIndent(infoComVersao, "", "  ")
	if err != nil {
		http.Error(w, fmt.Sprintf("Erro ao serializar dados: %v", err), http.StatusInternalServerError)
		return
	}

	if encriptado {
		// Criptografar os dados
		// Verificar se a chave pública existe
		currentDir, err := os.Getwd()
		if err != nil {
			errMsg := fmt.Sprintf("Erro ao obter diretório atual: %v", err)
			fmt.Println(errMsg)
			http.Error(w, errMsg, http.StatusInternalServerError)
			return
		}

		keysDir := filepath.Join(currentDir, "keys")
		publicKeyPath := filepath.Join(keysDir, "public_key.pem")

		if _, err := os.Stat(publicKeyPath); os.IsNotExist(err) {
			errMsg := fmt.Sprintf("Chave pública não encontrada em %s", publicKeyPath)
			fmt.Println(errMsg)
			http.Error(w, errMsg, http.StatusInternalServerError)
			return
		}

		// Criptografar os dados
		encryptedData, err := encryptWithPublicKey(jsonData)
		if err != nil {
			errMsg := fmt.Sprintf("Erro ao criptografar dados: %v", err)
			fmt.Println(errMsg)
			http.Error(w, errMsg, http.StatusInternalServerError)
			return
		}

		// Definir cabeçalhos e enviar resposta
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(encryptedData))
	} else {
		// Enviar JSON sem criptografia
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonData)
	}
}

// Handler para atualizar o IP do servidor de atualização
func updateServerIPHandler(w http.ResponseWriter, r *http.Request) {
	// Apenas aceitar requisições POST
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	// Ler o corpo da requisição
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Erro ao ler corpo da requisição", http.StatusBadRequest)
		return
	}

	// Estrutura para deserializar o JSON
	type UpdateRequest struct {
		IP    string `json:"ip_servidor"`
		Senha string `json:"senha"`
	}

	// Verificar se o corpo está vazio
	if len(body) == 0 {
		http.Error(w, "Corpo da requisição vazio", http.StatusBadRequest)
		return
	}

	// Tentar descriptografar o corpo com a chave pública
	decryptedData, err := decryptWithPublicKey(string(body))
	if err != nil {
		fmt.Printf("Erro ao descriptografar dados: %v\n", err)
		http.Error(w, "Erro ao descriptografar dados", http.StatusBadRequest)
		return
	}

	// Deserializar o JSON
	var request UpdateRequest
	err = json.Unmarshal([]byte(decryptedData), &request)
	if err != nil {
		http.Error(w, "Erro ao deserializar JSON", http.StatusBadRequest)
		return
	}

	// Verificar se o IP e a senha foram fornecidos
	if request.IP == "" {
		http.Error(w, "IP do servidor não fornecido", http.StatusBadRequest)
		return
	}

	if request.Senha == "" {
		http.Error(w, "Senha não fornecida", http.StatusBadRequest)
		return
	}

	// Verificar a senha (a senha correta seria verificada aqui)
	// Por segurança, usamos uma senha fixa que deve ser conhecida pelo servidor
	if request.Senha != "senha_secreta_do_agente" {
		http.Error(w, "Senha inválida", http.StatusUnauthorized)
		return
	}

	// Atualizar o IP no banco de dados
	err = updateServerIP(request.IP)
	if err != nil {
		fmt.Printf("Erro ao atualizar IP do servidor: %v\n", err)
		http.Error(w, "Erro ao atualizar IP do servidor", http.StatusInternalServerError)
		return
	}

	// Atualizar a variável global
	updateServerURL = request.IP

	// Responder com sucesso
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "IP do servidor de atualização alterado para: %s", request.IP)
	fmt.Printf("IP do servidor de atualização alterado para: %s\n", request.IP)
}

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
