package main

import (
	"bytes"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

// CommandPayload representa o payload para execução de comandos
type CommandPayload struct {
	Command string `json:"comando"`
	Type    string `json:"tipo"`  // "cmd" ou "ps" para PowerShell
	Senha   string `json:"senha"` // Senha para autenticação
}

// CommandResult representa o resultado da execução do comando
type CommandResult struct {
	Output   string `json:"saida"`
	ExitCode int    `json:"codigo_saida"`
	Error    string `json:"erro,omitempty"`
}

// commandHandler processa requisições para executar comandos no sistema
func commandHandler(w http.ResponseWriter, r *http.Request) {
	// Verificar se o método é POST
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	// Ler o corpo da requisição
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Erro ao ler corpo da requisição: %v", err), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Verificar se os dados estão criptografados
	var payload CommandPayload
	var decryptedData []byte

	// Tentar descriptografar os dados se estiverem em formato base64
	if _, err := base64.StdEncoding.DecodeString(string(body)); err == nil {
		// Obter o diretório do executável
		exePath, err := os.Executable()
		if err != nil {
			errMsg := fmt.Sprintf("Erro ao obter caminho do executável: %v", err)
			fmt.Println(errMsg)
			http.Error(w, errMsg, http.StatusInternalServerError)
			return
		}

		exeDir := filepath.Dir(exePath)
		keysDir := filepath.Join(exeDir, "keys")
		publicKeyPath := filepath.Join(keysDir, "public_key.pem")

		// Verificar se a chave pública existe
		if _, err := os.Stat(publicKeyPath); os.IsNotExist(err) {
			errMsg := fmt.Sprintf("Chave pública não encontrada em %s", publicKeyPath)
			fmt.Println(errMsg)
			http.Error(w, errMsg, http.StatusInternalServerError)
			return
		}

		// Verificar assinatura e extrair dados originais
		decryptedData, err = verifySignatureAndExtractData(string(body))
		if err != nil {
			errMsg := fmt.Sprintf("Erro ao verificar assinatura: %v", err)
			fmt.Println(errMsg)
			http.Error(w, errMsg, http.StatusBadRequest)
			return
		}
	} else {
		// Se não estiver criptografado, usar os dados brutos
		decryptedData = body
	}

	// Deserializar o payload JSON
	err = json.Unmarshal(decryptedData, &payload)
	if err != nil {
		http.Error(w, fmt.Sprintf("Erro ao deserializar payload: %v", err), http.StatusBadRequest)
		return
	}

	// Verificar a senha
	if payload.Senha != "senha_secreta_do_agente" {
		http.Error(w, "Senha inválida", http.StatusUnauthorized)
		return
	}

	// Verificar se o comando está vazio
	if payload.Command == "" {
		http.Error(w, "Comando vazio", http.StatusBadRequest)
		return
	}

	// Executar o comando
	var cmd *exec.Cmd
	if payload.Type == "ps" {
		// Comando PowerShell
		cmd = exec.Command("powershell", "-Command", payload.Command)
	} else {
		// Comando CMD (padrão)
		cmd = exec.Command("cmd", "/c", payload.Command)
	}

	// Capturar saída e erro
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Executar o comando
	err = cmd.Run()

	// Converter a saída para UTF-8 corretamente
	stdoutStr := stdout.String()
	stderrStr := stderr.String()

	// Tentar diferentes codificações para encontrar a que funciona melhor
	if stdoutStr != "" {
		// Primeiro tentar com CP850 (geralmente usado em CMD do Windows em português)
		reader := transform.NewReader(strings.NewReader(stdoutStr), charmap.CodePage850.NewDecoder())
		stdoutBytes, err := io.ReadAll(reader)
		if err == nil {
			stdoutStr = string(stdoutBytes)
		} else {
			// Se falhar, tentar com Windows-1252
			reader = transform.NewReader(strings.NewReader(stdoutStr), charmap.Windows1252.NewDecoder())
			stdoutBytes, err = io.ReadAll(reader)
			if err == nil {
				stdoutStr = string(stdoutBytes)
			}
		}
	}

	if stderrStr != "" {
		// Primeiro tentar com CP850
		reader := transform.NewReader(strings.NewReader(stderrStr), charmap.CodePage850.NewDecoder())
		stderrBytes, err := io.ReadAll(reader)
		if err == nil {
			stderrStr = string(stderrBytes)
		} else {
			// Se falhar, tentar com Windows-1252
			reader = transform.NewReader(strings.NewReader(stderrStr), charmap.Windows1252.NewDecoder())
			stderrBytes, err = io.ReadAll(reader)
			if err == nil {
				stderrStr = string(stderrBytes)
			}
		}
	}

	// Preparar o resultado
	result := CommandResult{
		Output:   stdoutStr,
		ExitCode: 0,
		Error:    stderrStr,
	}

	// Verificar se houve erro na execução
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitError.ExitCode()
		} else {
			result.Error = fmt.Sprintf("%v\n%s", err, stderrStr)
		}
	}

	// Serializar o resultado para JSON
	jsonResult, err := json.Marshal(result)
	if err != nil {
		http.Error(w, fmt.Sprintf("Erro ao serializar resultado: %v", err), http.StatusInternalServerError)
		return
	}

	// Verificar se deve criptografar a resposta
	if encriptado {
		// Criptografar os dados
		encryptedData, err := encryptWithPublicKey(jsonResult)
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
		w.Write(jsonResult)
	}
}

// verifySignatureAndExtractData verifica a assinatura e extrai os dados originais
func verifySignatureAndExtractData(encryptedData string) ([]byte, error) {
	// Decodificar o base64
	data, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return nil, fmt.Errorf("erro ao decodificar base64: %v", err)
	}

	// Obter o diretório do executável
	exePath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("erro ao obter caminho do executável: %v", err)
	}

	exeDir := filepath.Dir(exePath)
	// Caminho para a chave pública
	publicKeyPath := filepath.Join(exeDir, "keys", "public_key.pem")

	// Verificar se o arquivo existe
	if _, err := os.Stat(publicKeyPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("arquivo de chave pública não encontrado: %s", publicKeyPath)
	}

	// Carregar a chave pública
	publicKey, err := loadPublicKey(publicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("erro ao carregar chave pública: %v", err)
	}

	// Processar os chunks
	var originalData []byte
	i := 0

	for i < len(data) {
		// Ler o tamanho do chunk
		if i+4 >= len(data) {
			return nil, fmt.Errorf("formato inválido: dados truncados")
		}

		chunkLen := binary.BigEndian.Uint32(data[i : i+4])
		i += 4

		// Verificar o separador
		if i >= len(data) || data[i] != ':' {
			return nil, fmt.Errorf("formato inválido: separador não encontrado")
		}
		i++

		// Ler o chunk original + assinatura
		if i+int(chunkLen) > len(data) {
			return nil, fmt.Errorf("formato inválido: chunk truncado")
		}

		signedChunk := data[i : i+int(chunkLen)]
		i += int(chunkLen)

		// Verificar o separador final
		if i >= len(data) || data[i] != ':' {
			return nil, fmt.Errorf("formato inválido: separador final não encontrado")
		}
		i++

		// Separar os dados originais da assinatura
		// A assinatura tem tamanho fixo de 256 bytes para RSA-2048
		if len(signedChunk) <= 256 {
			return nil, fmt.Errorf("chunk muito pequeno para conter assinatura")
		}

		originalChunk := signedChunk[:len(signedChunk)-256]
		signature := signedChunk[len(signedChunk)-256:]

		// Verificar a assinatura
		hashed := sha256.Sum256(originalChunk)
		err = rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, hashed[:], signature)
		if err != nil {
			return nil, fmt.Errorf("assinatura inválida: %v", err)
		}

		// Adicionar os dados originais ao resultado
		originalData = append(originalData, originalChunk...)
	}

	return originalData, nil
}
