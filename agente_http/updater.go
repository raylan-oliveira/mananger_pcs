package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
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
	// Obter a versão atual
	currentVersion, err := getCurrentVersion()
	if err != nil {
		logUpdateError(fmt.Sprintf("Erro ao obter versão atual: %v", err))
		return false, "", err
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
	if latestVersion != currentVersion {
		return true, latestVersion, nil
	}

	return false, currentVersion, nil
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
func downloadAndUpdate(newVersion string) error {
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
