package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
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
			id INTEGER PRIMARY KEY AUTOINCREMENT,
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
		INSERT OR IGNORE INTO config (key, value) VALUES ('version', '1.0.0')
	`)
	if err != nil {
		return fmt.Errorf("erro ao inserir versão inicial: %v", err)
	}
	
	// Inserir IP de atualização padrão se não existir
	_, err = db.Exec(`
		INSERT OR IGNORE INTO config (key, value) VALUES ('ip_atualizacao', 'http://10.46.102.245:9991')
	`)
	if err != nil {
		return fmt.Errorf("erro ao inserir IP de atualização padrão: %v", err)
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

func getSystemInfoFromDB() (SystemInfo, error) {
	var info SystemInfo
	var infoJSON string
	
	// Obter a entrada mais recente
	err := db.QueryRow("SELECT info FROM system_info ORDER BY timestamp DESC LIMIT 1").Scan(&infoJSON)
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

func saveSystemInfoToDB(info SystemInfo) error {
	// Serializar struct para JSON
	infoJSON, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("erro ao serializar para JSON: %v", err)
	}
	
	// Inserir no banco de dados
	_, err = db.Exec("INSERT INTO system_info (info, timestamp) VALUES (?, ?)", string(infoJSON), time.Now())
	if err != nil {
		return fmt.Errorf("erro ao inserir no banco de dados: %v", err)
	}
	
	return nil
}

// Função para limpar o banco de dados
func clearDatabase() error {
	_, err := db.Exec("DELETE FROM system_info")
	if err != nil {
		return fmt.Errorf("erro ao limpar banco de dados: %v", err)
	}
	
	// Resetar o autoincrement
	_, err = db.Exec("DELETE FROM sqlite_sequence WHERE name='system_info'")
	if err != nil {
		return fmt.Errorf("erro ao resetar sequência: %v", err)
	}
	
	return nil
}

// Função para obter o IP do servidor de atualização
func getUpdateServerIP() (string, error) {
	var serverIP string
	err := db.QueryRow("SELECT value FROM config WHERE key = 'ip_atualizacao'").Scan(&serverIP)
	if err != nil {
		return "", fmt.Errorf("erro ao obter IP do servidor de atualização: %v", err)
	}
	return serverIP, nil
}

// Função para atualizar o IP do servidor de atualização
func updateServerIP(newIP string) error {
	_, err := db.Exec("UPDATE config SET value = ? WHERE key = 'ip_atualizacao'", newIP)
	if err != nil {
		return fmt.Errorf("erro ao atualizar IP do servidor: %v", err)
	}
	return nil
}