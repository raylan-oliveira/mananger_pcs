package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	// Replace go-sqlite3 with a pure Go implementation
	_ "modernc.org/sqlite"
)

var db *sql.DB

func initDatabase() error {
	var err error
	db, err = sql.Open("sqlite", "./system_info.db")
	if err != nil {
		return fmt.Errorf("erro ao abrir banco de dados: %v", err)
	}

	// Criar tabela system_info se não existir
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS system_info (
			id INTEGER PRIMARY KEY,
			info TEXT NOT NULL,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("erro ao criar tabela system_info: %v", err)
	}

	// Criar tabela config se não existir
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS config (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)
	`)
	if err != nil {
		return fmt.Errorf("erro ao criar tabela config: %v", err)
	}

	// Inserir versão inicial se não existir
	_, err = db.Exec(`
		INSERT OR IGNORE INTO config (key, value) VALUES ('version', '0.0.1')
	`)
	if err != nil {
		return fmt.Errorf("erro ao inserir versão inicial: %v", err)
	}

	// Inserir IP de atualização padrão se não existir
	_, err = db.Exec(`
		INSERT OR IGNORE INTO config (key, value) VALUES ('servidor_atualizacao', 'http://10.46.102.245:9991')
	`)
	if err != nil {
		return fmt.Errorf("erro ao inserir IP de atualização padrão: %v", err)
	}

	// Inserir intervalo de atualização de informações do sistema padrão se não existir
	_, err = db.Exec(`
		INSERT OR IGNORE INTO config (key, value) VALUES ('system_info_update_interval', '30')
	`)
	if err != nil {
		return fmt.Errorf("erro ao inserir intervalo de atualização padrão: %v", err)
	}

	// Inserir intervalo de verificação de atualizações padrão se não existir
	_, err = db.Exec(`
		INSERT OR IGNORE INTO config (key, value) VALUES ('update_check_interval', '30')
	`)
	if err != nil {
		return fmt.Errorf("erro ao inserir intervalo de verificação de atualizações padrão: %v", err)
	}

	return nil
}

// Função para obter a versão atual do aplicativo
func getCurrentVersion() (string, error) {
	var version string
	err := db.QueryRow("SELECT value FROM config WHERE key = 'version'").Scan(&version)
	if err != nil {
		return "", fmt.Errorf("erro ao obter versão atual: %v", err)
	}
	return version, nil
}

// Função para atualizar a versão do aplicativo
func updateVersion(newVersion string) error {
	_, err := db.Exec("UPDATE config SET value = ? WHERE key = 'version'", newVersion)
	if err != nil {
		return fmt.Errorf("erro ao atualizar versão: %v", err)
	}
	return nil
}

func closeDatabase() {
	if db != nil {
		db.Close()
	}
}

// getSystemInfoFromDB obtém as informações do sistema do banco de dados
func getSystemInfoFromDB() (SystemInfo, error) {
	var info SystemInfo
	var infoJSON string

	// Obter a linha com ID 1
	err := db.QueryRow("SELECT info FROM system_info WHERE id = 1").Scan(&infoJSON)
	if err != nil {
		return info, err
	}

	// Deserializar JSON para struct
	err = json.Unmarshal([]byte(infoJSON), &info)
	if err != nil {
		return info, fmt.Errorf("erro ao deserializar JSON: %v", err)
	}

	return info, nil
}

// saveSystemInfoToDB salva as informações do sistema no banco de dados
func saveSystemInfoToDB(info SystemInfo) error {
	// Serializar struct para JSON
	infoJSON, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("erro ao serializar para JSON: %v", err)
	}

	// Verificar se já existe uma linha na tabela
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM system_info").Scan(&count)
	if err != nil {
		return fmt.Errorf("erro ao verificar existência de dados: %v", err)
	}

	if count > 0 {
		// Atualizar a linha existente (usando o ID 1 assumindo que é a primeira linha)
		_, err = db.Exec("UPDATE system_info SET info = ?, timestamp = ? WHERE id = 1", string(infoJSON), time.Now())
		if err != nil {
			return fmt.Errorf("erro ao atualizar informações no banco de dados: %v", err)
		}
	} else {
		// Inserir nova linha com ID 1
		_, err = db.Exec("INSERT INTO system_info (id, info, timestamp) VALUES (1, ?, ?)", string(infoJSON), time.Now())
		if err != nil {
			return fmt.Errorf("erro ao inserir no banco de dados: %v", err)
		}
	}

	return nil
}

// clearDatabase limpa o banco de dados mantendo a estrutura
func clearDatabase() error {
	// Em vez de excluir todas as linhas, vamos apenas excluir linhas com ID diferente de 1
	_, err := db.Exec("DELETE FROM system_info WHERE id != 1")
	if err != nil {
		return fmt.Errorf("erro ao limpar linhas extras do banco de dados: %v", err)
	}

	// Verificar se existe uma linha com ID 1
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM system_info WHERE id = 1").Scan(&count)
	if err != nil {
		return fmt.Errorf("erro ao verificar existência de dados: %v", err)
	}

	// Se não existir uma linha com ID 1, não precisamos fazer nada
	// Se existir, vamos limpar seus dados
	if count > 0 {
		_, err = db.Exec("UPDATE system_info SET info = '{}', timestamp = ? WHERE id = 1", time.Now())
		if err != nil {
			return fmt.Errorf("erro ao limpar dados da linha principal: %v", err)
		}
	}

	return nil
}

// Função para obter o IP do servidor de atualização
func getUpdateServerIP() (string, error) {
	var serverIP string
	err := db.QueryRow("SELECT value FROM config WHERE key = 'servidor_atualizacao'").Scan(&serverIP)
	if err != nil {
		return "", fmt.Errorf("erro ao obter IP do servidor de atualização: %v", err)
	}
	return serverIP, nil
}

// Função para atualizar o IP do servidor de atualização
func updateServerIP(newIP string) error {
	_, err := db.Exec("UPDATE config SET value = ? WHERE key = 'servidor_atualizacao'", newIP)
	if err != nil {
		return fmt.Errorf("erro ao atualizar IP do servidor: %v", err)
	}
	return nil
}

// getSystemInfoUpdateInterval obtém o intervalo de atualização das informações do sistema em minutos
func getSystemInfoUpdateInterval() (int, error) {
	var intervalStr string
	err := db.QueryRow("SELECT value FROM config WHERE key = 'system_info_update_interval'").Scan(&intervalStr)
	if err != nil {
		return 10, fmt.Errorf("erro ao obter intervalo de atualização: %v", err)
	}

	// Converter string para inteiro
	interval, err := strconv.Atoi(intervalStr)
	if err != nil {
		return 10, fmt.Errorf("valor inválido para intervalo de atualização: %v", err)
	}

	// Garantir que o intervalo seja pelo menos 1 minuto
	if interval < 1 {
		interval = 1
	}

	return interval, nil
}

// getUpdateCheckInterval obtém o intervalo de verificação de atualizações em minutos
func getUpdateCheckInterval() (int, error) {
	var intervalStr string
	err := db.QueryRow("SELECT value FROM config WHERE key = 'update_check_interval'").Scan(&intervalStr)
	if err != nil {
		return 10, fmt.Errorf("erro ao obter intervalo de verificação de atualizações: %v", err)
	}

	// Converter string para inteiro
	interval, err := strconv.Atoi(intervalStr)
	if err != nil {
		return 10, fmt.Errorf("valor inválido para intervalo de verificação de atualizações: %v", err)
	}

	// Garantir que o intervalo seja pelo menos 1 minuto
	if interval < 1 {
		interval = 1
	}

	return interval, nil
}

// updateSystemInfoInterval atualiza o intervalo de atualização das informações do sistema
func updateSystemInfoInterval(minutes int) error {
	if minutes < 1 {
		minutes = 1 // Garantir que o intervalo seja pelo menos 1 minuto
	}

	_, err := db.Exec("UPDATE config SET value = ? WHERE key = 'system_info_update_interval'", strconv.Itoa(minutes))
	if err != nil {
		return fmt.Errorf("erro ao atualizar intervalo de atualização: %v", err)
	}

	return nil
}

// updateCheckInterval atualiza o intervalo de verificação de atualizações
func updateCheckInterval(minutes int) error {
	if minutes < 1 {
		minutes = 1 // Garantir que o intervalo seja pelo menos 1 minuto
	}

	_, err := db.Exec("UPDATE config SET value = ? WHERE key = 'update_check_interval'", strconv.Itoa(minutes))
	if err != nil {
		return fmt.Errorf("erro ao atualizar intervalo de verificação de atualizações: %v", err)
	}

	return nil
}
