package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// updateAgentServerIP envia uma requisição para atualizar o IP do servidor em um agente
func updateAgentServerIP(agentIP, newServerIP string) error {
	// Verificar se o agentIP inclui a porta
	if !strings.Contains(agentIP, ":") {
		agentIP = agentIP + ":9999" // Porta padrão do agente
	}
	
	// Verificar se o newServerIP começa com http://
	if !strings.HasPrefix(newServerIP, "http://") && !strings.HasPrefix(newServerIP, "https://") {
		newServerIP = "http://" + newServerIP
	}
	
	// Criar o payload
	type UpdatePayload struct {
		IP     string `json:"ip_servidor"`
		Senha  string `json:"senha"`
	}
	
	payload := UpdatePayload{
		IP:    newServerIP,
		Senha: "senha_secreta_do_agente", // Senha fixa conhecida pelo agente
	}
	
	// Serializar para JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("erro ao serializar payload: %v", err)
	}
	
	// Criptografar com a chave privada
	encryptedData, err := signWithPrivateKey(jsonData)
	if err != nil {
		return fmt.Errorf("erro ao criptografar dados: %v", err)
	}
	
	// Enviar a requisição para o agente
	url := fmt.Sprintf("http://%s/update-server", agentIP)
	resp, err := http.Post(url, "application/text", strings.NewReader(encryptedData))
	if err != nil {
		return fmt.Errorf("erro ao enviar requisição para o agente: %v", err)
	}
	defer resp.Body.Close()
	
	// Verificar o código de status
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("agente retornou código %d: %s", resp.StatusCode, string(bodyBytes))
	}
	
	return nil
}