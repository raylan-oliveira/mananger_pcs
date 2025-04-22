package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

var db *sql.DB

// initDatabase inicializa a conexão com o banco de dados
func initDatabase() error {
	// Obter o caminho do executável
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("erro ao obter caminho do executável: %v", err)
	}

	// Obter o diretório do executável
	exeDir := filepath.Dir(exePath)
	
	// Criar diretório data se não existir
	dataDir := filepath.Join(exeDir, "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("erro ao criar diretório data: %v", err)
	}
	
	// Caminho para o banco de dados (na pasta data)
	dbPath := filepath.Join(dataDir, "computers.db")
	
	// Abrir conexão com o banco de dados
	database, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("erro ao abrir banco de dados: %v", err)
	}

	// Configurar conexão
	database.SetMaxOpenConns(1)
	database.SetMaxIdleConns(1)
	database.SetConnMaxLifetime(time.Hour)

	// Verificar conexão
	if err := database.Ping(); err != nil {
		database.Close()
		return fmt.Errorf("erro ao conectar ao banco de dados: %v", err)
	}

	db = database
	return nil
}

// getAllAgentIPs retorna todos os IPs dos agentes ativos no banco de dados
func getAllAgentIPs() ([]string, error) {
	if db == nil {
		if err := initDatabase(); err != nil {
			return nil, err
		}
	}

	rows, err := db.Query(`
		SELECT ip_address 
		FROM computers 
		WHERE ip_address IS NOT NULL AND ip_address != ''
	`)
	if err != nil {
		return nil, fmt.Errorf("erro ao consultar IPs dos agentes: %v", err)
	}
	defer rows.Close()

	var ips []string
	for rows.Next() {
		var ip string
		if err := rows.Scan(&ip); err != nil {
			return nil, fmt.Errorf("erro ao ler IP do agente: %v", err)
		}
		ips = append(ips, ip)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erro ao iterar resultados: %v", err)
	}

	return ips, nil
}

// closeDatabase fecha a conexão com o banco de dados
func closeDatabase() {
	if db != nil {
		db.Close()
	}
}