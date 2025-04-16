package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// fileServerHandler é o manipulador personalizado para servir arquivos estáticos
func fileServerHandler(fileServer http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		
		// Obter o IP real do cliente
		clientIP := getClientIP(r)
		
		// Registrar conexão do cliente
		registerClient(clientIP)
		
		// Adicionar cabeçalhos para evitar cache
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		
		// Registrar informações da requisição
		log.Printf("Requisição: %s %s de %s", r.Method, r.URL.Path, clientIP)
		
		// Verificar se é uma requisição para a chave pública
		if r.URL.Path == "/public_key.pem" {
			// Redirecionar para o arquivo real na pasta keys
			r2 := new(http.Request)
			*r2 = *r
			r2.URL.Path = "/keys/public_key.pem"
			
			// Definir o tipo MIME correto para arquivos PEM
			w.Header().Set("Content-Type", "application/x-pem-file")
			w.Header().Set("Content-Disposition", "attachment; filename=\"public_key.pem\"")
			
			log.Printf("Servindo chave pública para %s", clientIP)
			fileServer.ServeHTTP(w, r2)
			
			// Registrar tempo de resposta
			log.Printf("Resposta: %s %s para %s - %v", r.Method, r.URL.Path, clientIP, time.Since(startTime))
			return
		}
		
		// Servir o arquivo
		fileServer.ServeHTTP(w, r)
		
		// Registrar tempo de resposta
		log.Printf("Resposta: %s %s para %s - %v", r.Method, r.URL.Path, clientIP, time.Since(startTime))
	}
}

// handleUpdateServerIP processa requisições para atualizar o IP do servidor
func handleUpdateServerIP(w http.ResponseWriter, r *http.Request) {
	// Apenas aceitar requisições POST
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	
	// Verificar se a chave privada está disponível
	if privateKey == nil {
		http.Error(w, "Funcionalidade indisponível: chave privada não carregada", http.StatusServiceUnavailable)
		return
	}
	
	// Obter o IP do cliente
	clientIP := getClientIP(r)
	log.Printf("Recebida solicitação para atualizar IP do servidor de %s", clientIP)
	
	// Ler o corpo da requisição
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Erro ao ler corpo da requisição", http.StatusBadRequest)
		return
	}
	
	// Estrutura para deserializar o JSON
	type UpdateRequest struct {
		TargetAgent string `json:"target_agent"`
		NewServerIP string `json:"new_server_ip"`
	}
	
	// Deserializar o JSON
	var request UpdateRequest
	err = json.Unmarshal(body, &request)
	if err != nil {
		http.Error(w, "Erro ao deserializar JSON", http.StatusBadRequest)
		return
	}
	
	// Verificar se os campos necessários foram fornecidos
	if request.TargetAgent == "" {
		http.Error(w, "IP do agente alvo não fornecido", http.StatusBadRequest)
		return
	}
	
	if request.NewServerIP == "" {
		http.Error(w, "Novo IP do servidor não fornecido", http.StatusBadRequest)
		return
	}
	
	// Atualizar o agente
	err = updateAgentServerIP(request.TargetAgent, request.NewServerIP)
	if err != nil {
		log.Printf("Erro ao atualizar agente %s: %v", request.TargetAgent, err)
		http.Error(w, fmt.Sprintf("Erro ao atualizar agente: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Responder com sucesso
	log.Printf("Agente %s atualizado com sucesso para usar o servidor %s", request.TargetAgent, request.NewServerIP)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Agente %s atualizado com sucesso", request.TargetAgent)
}