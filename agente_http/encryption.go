package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

// Função para carregar a chave pública
func loadPublicKey(path string) (*rsa.PublicKey, error) {
	// Ler o arquivo da chave pública
	pemData, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler arquivo de chave pública: %v", err)
	}
	
	// Decodificar o bloco PEM
	block, _ := pem.Decode(pemData)
	if block == nil || block.Type != "PUBLIC KEY" {
		return nil, fmt.Errorf("falha ao decodificar chave pública PEM")
	}
	
	// Analisar a chave pública
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("falha ao analisar chave pública: %v", err)
	}
	
	// Verificar se é uma chave RSA
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("chave não é uma chave pública RSA")
	}
	
	return rsaPub, nil
}

// Função para criptografar dados com a chave pública
func encryptWithPublicKey(data []byte) (string, error) {
	// Obter o diretório atual
	currentDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("erro ao obter diretório atual: %v", err)
	}
	
	// Caminho para a chave pública
	publicKeyPath := filepath.Join(currentDir, "keys", "public_key.pem")
	
	// Verificar se o arquivo existe
	if _, err := os.Stat(publicKeyPath); os.IsNotExist(err) {
		return "", fmt.Errorf("arquivo de chave pública não encontrado: %s", publicKeyPath)
	}
	
	// Carregar a chave pública
	publicKey, err := loadPublicKey(publicKeyPath)
	if err != nil {
		return "", fmt.Errorf("erro ao carregar chave pública: %v", err)
	}
	
	// Determinar o tamanho máximo que pode ser criptografado
	// RSA-2048 pode criptografar no máximo (2048/8) - 42 = 214 bytes por vez com OAEP e SHA-256
	maxSize := publicKey.Size() - 2*sha256.New().Size() - 2
	
	// Dividir os dados em chunks
	var encryptedChunks []byte
	
	for i := 0; i < len(data); i += maxSize {
		end := i + maxSize
		if end > len(data) {
			end = len(data)
		}
		
		chunk := data[i:end]

		// Criptografar o chunk
		encryptedChunk, err := rsa.EncryptOAEP(
			sha256.New(),
			rand.Reader,
			publicKey,
			chunk,
			nil,
		)
		if err != nil {
			return "", fmt.Errorf("erro ao criptografar chunk %d: %v", (i/maxSize)+1, err)
		}
		
		// Adicionar o tamanho do chunk e o separador
		chunkLen := make([]byte, 4)
		binary.BigEndian.PutUint32(chunkLen, uint32(len(encryptedChunk)))
		encryptedChunks = append(encryptedChunks, chunkLen...)
		encryptedChunks = append(encryptedChunks, ':')
		
		// Adicionar o chunk criptografado
		encryptedChunks = append(encryptedChunks, encryptedChunk...)
		encryptedChunks = append(encryptedChunks, ':')
	}
	
	// Codificar em base64 para transmissão
	encoded := base64.StdEncoding.EncodeToString(encryptedChunks)
	return encoded, nil
}

// Função para descriptografar dados com a chave pública
func decryptWithPublicKey(encryptedData string) (string, error) {
	// Decodificar o base64
	data, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return "", fmt.Errorf("erro ao decodificar base64: %v", err)
	}
	
	// Obter o diretório atual
	currentDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("erro ao obter diretório atual: %v", err)
	}
	
	// Caminho para a chave pública
	publicKeyPath := filepath.Join(currentDir, "keys", "public_key.pem")
	
	// Verificar se o arquivo existe
	if _, err := os.Stat(publicKeyPath); os.IsNotExist(err) {
		return "", fmt.Errorf("arquivo de chave pública não encontrado: %s", publicKeyPath)
	}
	
	// Carregar a chave pública
	publicKey, err := loadPublicKey(publicKeyPath)
	if err != nil {
		return "", fmt.Errorf("erro ao carregar chave pública: %v", err)
	}
	
	// Processar os chunks
	var decryptedData []byte
	i := 0
	
	for i < len(data) {
		// Ler o tamanho do chunk
		if i+4 >= len(data) {
			return "", fmt.Errorf("formato inválido: dados truncados")
		}
		
		chunkLen := binary.BigEndian.Uint32(data[i : i+4])
		i += 4
		
		// Verificar o separador
		if i >= len(data) || data[i] != ':' {
			return "", fmt.Errorf("formato inválido: separador não encontrado")
		}
		i++
		
		// Ler o chunk criptografado
		if i+int(chunkLen) > len(data) {
			return "", fmt.Errorf("formato inválido: chunk truncado")
		}
		
		encryptedChunk := data[i : i+int(chunkLen)]
		i += int(chunkLen)
		
		// Verificar o separador
		if i >= len(data) || data[i] != ':' {
			return "", fmt.Errorf("formato inválido: separador final não encontrado")
		}
		i++
		
		// Assumimos que os dados foram assinados com a chave privada do servidor
		// Vamos verificar se o chunk contém dados originais + assinatura
		// O formato esperado é: [dados originais][assinatura de 256 bytes]
		
		// Verificar se o chunk é grande o suficiente para conter uma assinatura
		if len(encryptedChunk) <= 256 {
			return "", fmt.Errorf("chunk muito pequeno para conter assinatura")
		}
		
		// Separar os dados originais e a assinatura
		originalData := encryptedChunk[:len(encryptedChunk)-256]
		signature := encryptedChunk[len(encryptedChunk)-256:]
		
		// Calcular o hash dos dados originais
		hashed := sha256.Sum256(originalData)
		
		// Verificar a assinatura
		err = rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, hashed[:], signature)
		if err != nil {
			return "", fmt.Errorf("erro ao verificar assinatura: %v", err)
		}
		
		// Se a assinatura for válida, adicionar os dados originais ao resultado
		decryptedData = append(decryptedData, originalData...)
	}
	
	return string(decryptedData), nil
}