package main

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Constantes para configuração do delay de atualização
const (
	// Delay mínimo em segundos (1 minuto)
	MinUpdateDelay = 60
	// Delay adicional máximo em segundos (2 minutos)
	MaxUpdateDelayAdd = 120
)

// logUpdateError registra erros de atualização em um arquivo de log
func logUpdateError(message string) {
	// Obter o caminho do executável
	exePath, err := os.Executable()
	if err != nil {
		fmt.Printf("Erro ao obter caminho do executável para log: %v\n", err)
		return
	}

	// Obter o diretório do executável
	exeDir := filepath.Dir(exePath)

	// Caminho do arquivo de log
	logPath := filepath.Join(exeDir, "update_errors.txt")

	// Abrir o arquivo para append (ou criar se não existir)
	logFile, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Erro ao abrir arquivo de log: %v\n", err)
		return
	}
	defer logFile.Close()

	// Formatar a mensagem com timestamp
	logMessage := fmt.Sprintf("[%s] %s\n", time.Now().Format("2006-01-02 15:04:05"), message)

	// Escrever no arquivo
	if _, err := logFile.WriteString(logMessage); err != nil {
		fmt.Printf("Erro ao escrever no arquivo de log: %v\n", err)
	}

	// Também exibir no console para facilitar o diagnóstico
	fmt.Println(message)
}

// checkForUpdates verifica se há atualizações disponíveis
func checkForUpdates() (bool, string, error) {
	// Verificar se estamos em um processo de atualização recente
	if isRecentlyUpdated() {
		logUpdateError("Atualização recente detectada, pulando verificação de atualizações")
		return false, "", nil
	}

	// Obter a versão atual
	currentVersion, err := getCurrentVersion()
	if err != nil {
		logUpdateError(fmt.Sprintf("Erro ao obter versão atual: %v", err))
		return false, "", err
	}

	// Verificar se o arquivo de versão local existe e é recente
	exePath, _ := os.Executable()
	exeDir := filepath.Dir(exePath)
	versionPath := filepath.Join(exeDir, "version.txt")

	// Se o arquivo version.txt existir e for recente (menos de 5 minutos), não verificar atualizações
	if info, err := os.Stat(versionPath); err == nil {
		if time.Since(info.ModTime()) < 5*time.Minute {
			logUpdateError("Arquivo version.txt recente encontrado, pulando verificação de atualizações")
			return false, currentVersion, nil
		}
	}

	// URL do arquivo de versão
	versionURL := updateServerURL + "/version.txt"

	// Fazer requisição HTTP para obter a versão mais recente
	// Adicionar timeout para evitar bloqueio indefinido
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	resp, err := client.Get(versionURL)
	if err != nil {
		errMsg := fmt.Sprintf("Erro ao acessar servidor de atualizações: %v", err)
		logUpdateError(errMsg)
		return false, "", fmt.Errorf(errMsg)
	}
	defer resp.Body.Close()

	// Verificar o código de status da resposta
	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("Servidor retornou código de status %d", resp.StatusCode)
		logUpdateError(errMsg)
		return false, "", fmt.Errorf(errMsg)
	}

	// Ler o conteúdo da resposta
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, "", fmt.Errorf("erro ao ler resposta do servidor: %v", err)
	}

	// Obter a versão mais recente (remover espaços e quebras de linha)
	latestVersion := strings.TrimSpace(string(body))

	// Verificar se a versão está no formato esperado (x.y.z)
	if !isValidVersionFormat(latestVersion) {
		return false, "", fmt.Errorf("formato de versão inválido: %s", latestVersion)
	}
	
	// Comparar versões
	if compareVersions(latestVersion, currentVersion) > 0 {
		return true, latestVersion, nil
	}

	return false, currentVersion, nil
}

// compareVersions compara duas versões no formato x.y.z
// Retorna:
// -1 se v1 < v2
//
//	0 se v1 == v2
//	1 se v1 > v2
func compareVersions(v1, v2 string) int {
	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	// Garantir que ambas as versões têm 3 partes
	if len(parts1) != 3 || len(parts2) != 3 {
		// Se alguma versão não tiver 3 partes, considerar iguais para evitar atualizações desnecessárias
		return 0
	}

	// Comparar cada parte da versão
	for i := 0; i < 3; i++ {
		// Converter para inteiros
		n1, err1 := strconv.Atoi(parts1[i])
		n2, err2 := strconv.Atoi(parts2[i])

		// Se houver erro na conversão, considerar iguais
		if err1 != nil || err2 != nil {
			continue
		}

		// Comparar os números
		if n1 < n2 {
			return -1
		} else if n1 > n2 {
			return 1
		}
	}

	// Se chegou aqui, as versões são iguais
	return 0
}

// isRecentlyUpdated verifica se o aplicativo foi atualizado recentemente
func isRecentlyUpdated() bool {
	// Verificar se existe um arquivo de marcação de atualização recente
	exePath, err := os.Executable()
	if err != nil {
		return false
	}

	exeDir := filepath.Dir(exePath)
	updateFlagPath := filepath.Join(exeDir, ".recent_update")

	// Verificar se o arquivo existe
	info, err := os.Stat(updateFlagPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
		// Em caso de outro erro, assumimos que não foi atualizado recentemente
		return false
	}

	// Verificar se o arquivo foi criado nas últimas 5 minutos (aumentado de 2 para 5)
	return time.Since(info.ModTime()) < 5*time.Minute
}

// isValidVersionFormat verifica se a string está no formato de versão esperado (x.y.z)
func isValidVersionFormat(version string) bool {
	// Verificar se a versão corresponde ao padrão x.y.z onde x, y e z são números
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return false
	}

	// Verificar se cada parte é um número
	for _, part := range parts {
		if len(part) == 0 {
			return false
		}
		for _, char := range part {
			if char < '0' || char > '9' {
				return false
			}
		}
	}

	return true
}

// downloadAndUpdate baixa e instala a atualização
func downloadAndUpdate(newVersion string, isInitialCheck bool) error {
	// Adicionar delay apenas se não for verificação inicial
	if !isInitialCheck {
		delaySeconds := time.Duration(MinUpdateDelay + rand.Intn(MaxUpdateDelayAdd)) // Gera um número entre 60 e 180 segundos
		logUpdateError(fmt.Sprintf("Aguardando %d segundos antes de baixar a atualização...", delaySeconds))
		time.Sleep(delaySeconds * time.Second)
	}

	// Obter o caminho do executável atual
	exePath, err := os.Executable()
	if err != nil {
		errMsg := fmt.Sprintf("Erro ao obter caminho do executável: %v", err)
		logUpdateError(errMsg)
		return fmt.Errorf(errMsg)
	}

	// Garantir que temos o caminho absoluto
	exePath, err = filepath.Abs(exePath)
	if err != nil {
		errMsg := fmt.Sprintf("Erro ao obter caminho absoluto do executável: %v", err)
		logUpdateError(errMsg)
		return fmt.Errorf(errMsg)
	}

	// Obter o diretório do executável
	exeDir := filepath.Dir(exePath)
	logUpdateError(fmt.Sprintf("Diretório do executável: %s", exeDir))

	// Definir caminhos para os arquivos
	backupPath := filepath.Join(exeDir, "agente_http.exe~")
	newExePath := filepath.Join(exeDir, "agente_http.exe")
	versionPath := filepath.Join(exeDir, "version.txt")

	// Verificar se já existe um backup e removê-lo se necessário
	if _, err := os.Stat(backupPath); err == nil {
		logUpdateError("Removendo backup antigo...")
		err = os.Remove(backupPath)
		if err != nil {
			logUpdateError(fmt.Sprintf("Aviso: Não foi possível remover backup antigo: %v", err))
			// Continuar mesmo com erro
		}
	}

	// 1. Renomear o executável atual para backup
	logUpdateError(fmt.Sprintf("Renomeando executável atual para backup: %s -> %s", exePath, backupPath))
	err = os.Rename(exePath, backupPath)
	if err != nil {
		errMsg := fmt.Sprintf("Erro ao renomear executável atual: %v", err)
		logUpdateError(errMsg)
		return fmt.Errorf(errMsg)
	}

	// 2. Baixar a nova versão do executável
	logUpdateError(fmt.Sprintf("Baixando nova versão do executável de %s", updateServerURL+"/agente_http.exe"))
	err = downloadFile(updateServerURL+"/agente_http.exe", newExePath)
	if err != nil {
		// Restaurar o executável original em caso de erro
		logUpdateError(fmt.Sprintf("Erro ao baixar nova versão: %v. Restaurando executável original...", err))
		os.Rename(backupPath, exePath)
		return err
	}

	// 3. Baixar a chave pública atualizada
	logUpdateError("Baixando chave pública atualizada...")
	keysDir := filepath.Join(exeDir, "keys")
	if _, err := os.Stat(keysDir); os.IsNotExist(err) {
		err = os.MkdirAll(keysDir, 0700)
		if err != nil {
			logUpdateError(fmt.Sprintf("Aviso: Não foi possível criar diretório de chaves: %v", err))
		}
	}

	publicKeyPath := filepath.Join(keysDir, "public_key.pem")
	err = downloadFile(updateServerURL+"/public_key.pem", publicKeyPath)
	if err != nil {
		logUpdateError(fmt.Sprintf("Aviso: Não foi possível baixar chave pública: %v", err))
		// Continuar mesmo com erro na chave pública
	}

	// 4. Baixar o arquivo version.txt
	logUpdateError("Baixando arquivo de versão...")
	err = downloadFile(updateServerURL+"/version.txt", versionPath)
	if err != nil {
		logUpdateError(fmt.Sprintf("Aviso: Não foi possível baixar arquivo de versão: %v", err))
		// Criar o arquivo manualmente com a versão informada
		err = os.WriteFile(versionPath, []byte(newVersion), 0644)
		if err != nil {
			logUpdateError(fmt.Sprintf("Aviso: Não foi possível criar arquivo de versão: %v", err))
		}
	}

	// Marcar que uma atualização foi realizada recentemente
	err = markRecentUpdate()
	if err != nil {
		logUpdateError(fmt.Sprintf("Aviso: Não foi possível marcar atualização recente: %v", err))
	}

	// 5. Fechar o servidor HTTP para liberar a porta
	logUpdateError("Fechando servidor HTTP para liberar a porta...")
	// Implementado em http_server.go - será chamado antes de iniciar o novo executável

	// 7. Executar o novo executável
	logUpdateError("Iniciando nova versão do aplicativo...")
	cmd := exec.Command(newExePath)
	cmd.Dir = exeDir
	err = cmd.Start()
	if err != nil {
		errMsg := fmt.Sprintf("Erro ao iniciar nova versão: %v. Restaurando executável original...", err)
		logUpdateError(errMsg)
		os.Rename(backupPath, exePath)
		return fmt.Errorf(errMsg)
	}

	// 8. Fechar o executável atual (será feito pelo chamador)
	logUpdateError("Nova versão iniciada com sucesso. Encerrando versão atual...")

	return nil
}

// markRecentUpdate marca que uma atualização foi realizada recentemente
func markRecentUpdate() error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}

	exeDir := filepath.Dir(exePath)
	updateFlagPath := filepath.Join(exeDir, ".recent_update")

	// Criar arquivo de marcação
	return os.WriteFile(updateFlagPath, []byte(time.Now().String()), 0644)
}

// Função auxiliar para baixar arquivos com verificações
func downloadFile(url, destPath string) error {
	// Criar cliente HTTP com timeout
	client := &http.Client{
		Timeout: 5 * time.Minute,
	}

	// Fazer requisição HTTP
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("erro na requisição HTTP: %v", err)
	}
	defer resp.Body.Close()

	// Verificar código de status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("servidor retornou código %d", resp.StatusCode)
	}

	// Criar arquivo de destino
	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("erro ao criar arquivo: %v", err)
	}
	defer out.Close()

	// Copiar conteúdo para o arquivo
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("erro ao salvar arquivo: %v", err)
	}

	return nil
}

// restartApplication reinicia o aplicativo após a atualização
func restartApplication() {
	logUpdateError("Reiniciando aplicativo após atualização...")

	// Encerrar o servidor HTTP para liberar a porta
	shutdownHTTPServer()

	// Encerrar o processo atual
	os.Exit(0)
}
