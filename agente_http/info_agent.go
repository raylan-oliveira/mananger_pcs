package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// getAgentInfo obtém as informações do agente a partir do banco de dados
func getAgentInfo() AgenteInfo {
	// Obter a versão atual do agente do banco de dados
	versaoAgente, err := getCurrentVersion()
	if err != nil {
		fmt.Printf("Erro ao obter versão do agente: %v\n", err)
		versaoAgente = "desconhecida"
	}

	// Obter o IP do servidor de atualização
	servidorAtualizacao, err := getUpdateServerIP()
	if err != nil {
		fmt.Printf("Erro ao obter servidor de atualização: %v\n", err)
		servidorAtualizacao = "desconhecido"
	}

	// Obter os intervalos de atualização
	systemInfoUpdateInterval, err := getSystemInfoUpdateInterval()
	if err != nil {
		fmt.Printf("Erro ao obter intervalo de atualização: %v\n", err)
		systemInfoUpdateInterval = 10
	}

	updateCheckInterval, err := getUpdateCheckInterval()
	if err != nil {
		fmt.Printf("Erro ao obter intervalo de verificação: %v\n", err)
		updateCheckInterval = 10
	}

	// Criar e retornar o objeto AgenteInfo
	return AgenteInfo{
		VersaoAgente:             versaoAgente,
		ServidorAtualizacao:      servidorAtualizacao,
		SystemInfoUpdateInterval: fmt.Sprintf("%d", systemInfoUpdateInterval),
		UpdateCheckInterval:      fmt.Sprintf("%d", updateCheckInterval),
	}
}

// updateAgentVersion atualiza a versão do agente no banco de dados a partir do arquivo version.txt
func updateAgentVersion() error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("erro ao obter caminho do executável: %v", err)
	}

	exeDir := filepath.Dir(exePath)
	versionPath := filepath.Join(exeDir, "version.txt")

	// Verificar se o arquivo existe
	if _, err := os.Stat(versionPath); os.IsNotExist(err) {
		return fmt.Errorf("arquivo de versão não encontrado: %s", versionPath)
	}

	// Ler o conteúdo do arquivo
	content, err := os.ReadFile(versionPath)
	if err != nil {
		return fmt.Errorf("erro ao ler arquivo de versão: %v", err)
	}

	// Atualizar a versão no banco de dados
	version := string(content)
	err = updateVersion(version)
	if err != nil {
		return fmt.Errorf("erro ao atualizar versão no banco de dados: %v", err)
	}

	return nil
}