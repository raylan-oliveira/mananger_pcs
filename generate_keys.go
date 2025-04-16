package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	fmt.Println("=== Gerador de Chaves RSA ===")
	
	// Criando diretório para as chaves no diretório atual
	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("Erro ao obter diretório atual: %v\n", err)
		return
	}
	
	keysDir := filepath.Join(currentDir, "keys")
	if _, err := os.Stat(keysDir); os.IsNotExist(err) {
		err = os.MkdirAll(keysDir, 0700)
		if err != nil {
			fmt.Printf("Erro ao criar diretório de chaves: %v\n", err)
			return
		}
		fmt.Printf("Diretório de chaves criado: %s\n", keysDir)
	}
	
	// Gerando par de chaves RSA
	fmt.Println("Gerando par de chaves RSA (2048 bits)...")
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		fmt.Printf("Erro ao gerar chaves RSA: %v\n", err)
		return
	}
	
	// Salvando a chave privada
	privateKeyPath := filepath.Join(keysDir, "private_key.pem")
	privateKeyFile, err := os.Create(privateKeyPath)
	if err != nil {
		fmt.Printf("Erro ao criar arquivo de chave privada: %v\n", err)
		return
	}
	defer privateKeyFile.Close()
	
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	err = pem.Encode(privateKeyFile, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	})
	if err != nil {
		fmt.Printf("Erro ao salvar chave privada: %v\n", err)
		return
	}
	fmt.Printf("Chave privada salva em: %s\n", privateKeyPath)
	
	// Salvando a chave pública
	publicKeyPath := filepath.Join(keysDir, "public_key.pem")
	publicKeyFile, err := os.Create(publicKeyPath)
	if err != nil {
		fmt.Printf("Erro ao criar arquivo de chave pública: %v\n", err)
		return
	}
	defer publicKeyFile.Close()
	
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		fmt.Printf("Erro ao serializar chave pública: %v\n", err)
		return
	}
	
	err = pem.Encode(publicKeyFile, &pem.Block{
		Type:  "PUBLIC KEY",  // Changed from "RSA PUBLIC KEY" to "PUBLIC KEY"
		Bytes: publicKeyBytes,
	})
	if err != nil {
		fmt.Printf("Erro ao salvar chave pública: %v\n", err)
		return
	}
	fmt.Printf("Chave pública salva em: %s\n", publicKeyPath)
	
	fmt.Println("\nPar de chaves gerado com sucesso!")
	fmt.Println("Distribua a chave pública (public_key.pem) para todos os agentes.")
	fmt.Println("Mantenha a chave privada (private_key.pem) apenas no servidor.")
}