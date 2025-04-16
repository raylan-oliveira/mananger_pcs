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
	"os"
	"path/filepath"
)

// loadPrivateKey carrega a chave privada RSA de um arquivo PEM
func loadPrivateKey(path string) (*rsa.PrivateKey, error) {
	// Verificar se o arquivo existe
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Criar o diretório se não existir
		dir := filepath.Dir(path)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			if err := os.MkdirAll(dir, 0700); err != nil {
				return nil, fmt.Errorf("erro ao criar diretório para chaves: %v", err)
			}
		}
		return nil, fmt.Errorf("arquivo de chave privada não encontrado: %s", path)
	}
	
	// Ler o arquivo da chave privada
	pemData, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler arquivo de chave privada: %v", err)
	}
	
	// Decodificar o bloco PEM
	block, _ := pem.Decode(pemData)
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		return nil, fmt.Errorf("falha ao decodificar chave privada PEM")
	}
	
	// Analisar a chave privada
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("falha ao analisar chave privada: %v", err)
	}
	
	return privateKey, nil
}

// signWithPrivateKey assina dados com a chave privada
func signWithPrivateKey(data []byte) (string, error) {
	// Determinar o tamanho máximo que pode ser assinado
	maxSize := privateKey.Size() - 2*sha256.New().Size() - 2
	
	// Dividir os dados em chunks
	var signedChunks []byte
	
	for i := 0; i < len(data); i += maxSize {
		end := i + maxSize
		if end > len(data) {
			end = len(data)
		}
		
		chunk := data[i:end]
		
		// Calcular o hash do chunk
		hashed := sha256.Sum256(chunk)
		
		// Assinar o hash
		signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hashed[:])
		if err != nil {
			return "", fmt.Errorf("erro ao assinar chunk %d: %v", (i/maxSize)+1, err)
		}
		
		// Adicionar o tamanho do chunk e o separador
		chunkLen := make([]byte, 4)
		binary.BigEndian.PutUint32(chunkLen, uint32(len(chunk)+len(signature)))
		signedChunks = append(signedChunks, chunkLen...)
		signedChunks = append(signedChunks, ':')
		
		// Adicionar o chunk original
		signedChunks = append(signedChunks, chunk...)
		
		// Adicionar a assinatura
		signedChunks = append(signedChunks, signature...)
		signedChunks = append(signedChunks, ':')
	}
	
	// Codificar em base64 para transmissão
	encoded := base64.StdEncoding.EncodeToString(signedChunks)
	return encoded, nil
}