package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
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

// Aqui você pode adicionar novos comandos para os agentes
// Por exemplo:

/*
func restartAgent(agentIP string) error {
    // Implementação para reiniciar o agente
}

func getAgentStatus(agentIP string) (string, error) {
    // Implementação para obter o status do agente
}

func updateAgentConfig(agentIP string, config map[string]interface{}) error {
    // Implementação para atualizar a configuração do agente
}
*/

// getAgentInfo obtém informações detalhadas de um agente
func getAgentInfo(agentIP string, timeout int) (map[string]interface{}, error) {
	// Verificar se o agentIP inclui a porta
	if !strings.Contains(agentIP, ":") {
		agentIP = agentIP + ":9999" // Porta padrão do agente
	}
	
	// Configurar cliente HTTP com timeout
	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}
	
	// Solicitar dados criptografados
	url := fmt.Sprintf("http://%s?encrypt=true", agentIP)
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("erro ao conectar com o agente: %v", err)
	}
	defer resp.Body.Close()
	
	// Verificar o código de status
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("agente retornou código %d: %s", resp.StatusCode, string(bodyBytes))
	}
	
	// Ler o corpo da resposta
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler resposta: %v", err)
	}
	
	// Verificar se a resposta está criptografada
	var result map[string]interface{}
	
	// Tentar decodificar como JSON primeiro
	err = json.Unmarshal(body, &result)
	if err == nil {
		// Resposta já está em formato JSON
		return result, nil
	}
	
	// Tentar descriptografar a resposta
	decryptedData, err := decryptData(body)
	if err != nil {
		return nil, fmt.Errorf("erro ao processar resposta: %v", err)
	}
	
	return decryptedData, nil
}

// decryptData descriptografa dados usando a chave privada
func decryptData(encryptedData []byte) (map[string]interface{}, error) {
	// Decodificando de base64
	encryptedBytes, err := base64.StdEncoding.DecodeString(string(encryptedData))
	if err != nil {
		return nil, fmt.Errorf("erro ao decodificar base64: %v", err)
	}
	
	// Separando os chunks criptografados
	var chunks [][]byte
	i := 0
	for i < len(encryptedBytes) {
		// Lendo o tamanho do chunk
		if i+4 >= len(encryptedBytes) {
			return nil, fmt.Errorf("formato inválido: tamanho do chunk não encontrado")
		}
		
		chunkLen := binary.BigEndian.Uint32(encryptedBytes[i : i+4])
		i += 4
		
		// Pulando o separador ':'
		if i >= len(encryptedBytes) || encryptedBytes[i] != ':' {
			return nil, fmt.Errorf("formato inválido: separador não encontrado")
		}
		i++
		
		// Lendo o chunk
		if i+int(chunkLen) > len(encryptedBytes) {
			return nil, fmt.Errorf("formato inválido: chunk incompleto")
		}
		
		chunk := encryptedBytes[i : i+int(chunkLen)]
		chunks = append(chunks, chunk)
		i += int(chunkLen)
		
		// Pulando o separador ':'
		if i < len(encryptedBytes) && encryptedBytes[i] == ':' {
			i++
		}
	}
	
	// Descriptografando cada chunk
	var decryptedData []byte
	
	// Verificar se os chunks contêm dados assinados ou criptografados
	if len(chunks) > 0 && len(chunks[0]) > privateKey.Size() {
		// Provavelmente são dados assinados, não criptografados
		// Formato: [dados originais][assinatura]
		for _, chunk := range chunks {
			dataLen := len(chunk) - privateKey.Size()
			if dataLen <= 0 {
				continue
			}
			
			originalData := chunk[:dataLen]
			decryptedData = append(decryptedData, originalData...)
		}
	} else {
		// Dados criptografados com OAEP
		for _, chunk := range chunks {
			decryptedChunk, err := rsa.DecryptOAEP(
				sha256.New(),
				rand.Reader,
				privateKey,
				chunk,
				nil,
			)
			if err != nil {
				return nil, fmt.Errorf("erro ao descriptografar chunk: %v", err)
			}
			decryptedData = append(decryptedData, decryptedChunk...)
		}
	}
	
	// Convertendo para JSON
	var result map[string]interface{}
	err = json.Unmarshal(decryptedData, &result)
	if err != nil {
		return nil, fmt.Errorf("erro ao converter JSON: %v", err)
	}
	
	return result, nil
}