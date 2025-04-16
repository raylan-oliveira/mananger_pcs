package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

// Função para descriptografar dados usando a chave privada
func decryptData(encryptedData []byte) (map[string]interface{}, error) {
	// Decodificando de base64
	encryptedBytes, err := base64.StdEncoding.DecodeString(string(encryptedData))
	if err != nil {
		return nil, fmt.Errorf("erro ao decodificar base64: %v", err)
	}
	
	// Carregando a chave privada
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("erro ao obter diretório atual: %v", err)
	}
	
	keysDir := filepath.Join(currentDir, "keys")
	privateKeyPath := filepath.Join(keysDir, "private_key.pem")
	
	privateKeyBytes, err := ioutil.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler arquivo de chave privada: %v", err)
	}
	
	block, _ := pem.Decode(privateKeyBytes)
	if block == nil {
		return nil, fmt.Errorf("falha ao decodificar chave privada PEM")
	}
	
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("falha ao analisar chave privada: %v", err)
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
		if i >= len(encryptedBytes) || encryptedBytes[i] != ':' {
			return nil, fmt.Errorf("formato inválido: separador não encontrado após chunk")
		}
		i++
	}
	
	// Descriptografando cada chunk
	var decryptedData []byte
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
	
	// Convertendo para JSON
	var result map[string]interface{}
	err = json.Unmarshal(decryptedData, &result)
	if err != nil {
		return nil, fmt.Errorf("erro ao converter JSON: %v", err)
	}
	
	return result, nil
}