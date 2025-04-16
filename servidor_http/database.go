package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	// Using pure Go SQLite implementation instead of CGO-based one
	_ "modernc.org/sqlite"
)

var db *sql.DB

// Inicializa o banco de dados
func initDatabase() error {
	// Verificar se o diretório de dados existe
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("erro ao obter diretório atual: %v", err)
	}

	dataDir := filepath.Join(currentDir, "data")
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		// Criar diretório se não existir
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			return fmt.Errorf("erro ao criar diretório de dados: %v", err)
		}
	}

	// Abrir conexão com o banco de dados (usando sqlite em vez de sqlite3)
	dbPath := filepath.Join(dataDir, "computers.db")
	database, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("erro ao abrir banco de dados: %v", err)
	}

	// Definir configurações de conexão
	database.SetMaxOpenConns(1)
	database.SetMaxIdleConns(1)
	database.SetConnMaxLifetime(time.Hour)

	// Verificar conexão
	if err := database.Ping(); err != nil {
		database.Close()
		return fmt.Errorf("erro ao conectar ao banco de dados: %v", err)
	}

	// Criar tabelas se não existirem
	if err := createTables(database); err != nil {
		database.Close()
		return fmt.Errorf("erro ao criar tabelas: %v", err)
	}

	db = database
	return nil
}

// Cria as tabelas necessárias no banco de dados
func createTables(db *sql.DB) error {
	// Tabela principal de computadores
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS computers (
			mac_address TEXT PRIMARY KEY,
			hostname TEXT,
			ip_address TEXT,
			os_name TEXT,
			cpu_model TEXT,
			ram_total REAL,
			disk_c_total REAL,
			agent_version TEXT,
			last_seen TIMESTAMP,
			first_seen TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("erro ao criar tabela computers: %v", err)
	}

	// Tabela para armazenar o histórico de dados completos em JSON
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS computer_data (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			mac_address TEXT,
			data_json TEXT,
			timestamp TIMESTAMP,
			FOREIGN KEY (mac_address) REFERENCES computers(mac_address)
		)
	`)
	if err != nil {
		return fmt.Errorf("erro ao criar tabela computer_data: %v", err)
	}

	return nil
}

// Extrai o MAC da primeira interface de rede ativa
func extractPrimaryMacAddress(info map[string]interface{}) (string, error) {
	// Verificar se existe informação de rede
	redeInfo, ok := info["rede"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("informações de rede não encontradas")
	}

	// Verificar se existem interfaces
	interfaces, ok := redeInfo["interfaces"].([]interface{})
	if !ok || len(interfaces) == 0 {
		return "", fmt.Errorf("interfaces de rede não encontradas")
	}

	// Procurar pela primeira interface com status "Up"
	for _, iface := range interfaces {
		ifaceMap, ok := iface.(map[string]interface{})
		if !ok {
			continue
		}

		// Verificar status
		status, ok := ifaceMap["status"].(string)
		if !ok {
			continue
		}

		// Verificar se está ativa (status contém "Up")
		if strings.Contains(status, "Up") {
			// Obter MAC address
			mac, ok := ifaceMap["mac"].(string)
			if !ok || mac == "" {
				continue
			}
			return mac, nil
		}
	}

	return "", fmt.Errorf("nenhuma interface de rede ativa encontrada")
}

// Salva ou atualiza informações do computador no banco de dados
func saveComputerInfo(info map[string]interface{}, ip string) error {
	// Extrair MAC address primário
	macAddress, err := extractPrimaryMacAddress(info)
	if err != nil {
		return fmt.Errorf("erro ao extrair MAC address: %v", err)
	}

	// Verificar se o MAC é válido
	if macAddress == "" {
		return fmt.Errorf("MAC address inválido ou vazio")
	}

	// Extrair outros dados
	var hostname, osName, cpuModel, agentVersion string
	var ramTotal, diskCTotal float64

	// Extrair hostname
	if sistema, ok := info["sistema"].(map[string]interface{}); ok {
		if host, ok := sistema["nome_host"].(string); ok {
			hostname = host
		}
		if os, ok := sistema["nome_so"].(string); ok {
			osName = os
		}
	}

	// Extrair CPU
	if cpu, ok := info["cpu"].(map[string]interface{}); ok {
		if model, ok := cpu["modelo"].(string); ok {
			cpuModel = model
		}
	}

	// Extrair RAM
	if memoria, ok := info["memoria"].(map[string]interface{}); ok {
		if total, ok := memoria["total_gb"].(float64); ok {
			ramTotal = total
		}
	}

	// Extrair disco C
	if discos, ok := info["discos"].([]interface{}); ok && len(discos) > 0 {
		for _, d := range discos {
			disco, ok := d.(map[string]interface{})
			if !ok {
				continue
			}
			
			if dispositivo, ok := disco["dispositivo"].(string); ok && dispositivo == "C:" {
				if total, ok := disco["total_gb"].(float64); ok {
					diskCTotal = total
				}
				break
			}
		}
	}

	// Extrair versão do agente
	if agente, ok := info["agente"].(map[string]interface{}); ok {
		if versao, ok := agente["versao_agente"].(string); ok {
			agentVersion = versao
		}
	}

	now := time.Now()

	// Verificar se o computador já existe usando uma transação para garantir consistência
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("erro ao iniciar transação: %v", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	var exists bool
	err = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM computers WHERE mac_address = ?)", macAddress).Scan(&exists)
	if err != nil {
		return fmt.Errorf("erro ao verificar existência do computador: %v", err)
	}

	if exists {
		// Atualizar registro existente
		_, err = tx.Exec(`
			UPDATE computers 
			SET hostname = ?, ip_address = ?, os_name = ?, cpu_model = ?, 
				ram_total = ?, disk_c_total = ?, agent_version = ?, last_seen = ?
			WHERE mac_address = ?
		`, hostname, ip, osName, cpuModel, ramTotal, diskCTotal, agentVersion, now, macAddress)
		if err != nil {
			return fmt.Errorf("erro ao atualizar computador: %v", err)
		}
	} else {
		// Inserir novo registro
		_, err = tx.Exec(`
			INSERT INTO computers 
			(mac_address, hostname, ip_address, os_name, cpu_model, ram_total, disk_c_total, agent_version, last_seen, first_seen)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, macAddress, hostname, ip, osName, cpuModel, ramTotal, diskCTotal, agentVersion, now, now)
		if err != nil {
			return fmt.Errorf("erro ao inserir computador: %v", err)
		}
	}

	// Salvar dados completos em JSON
	jsonData, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("erro ao serializar dados JSON: %v", err)
	}

	// Verificar se já existe um registro na tabela computer_data para este MAC
	var dataExists bool
	err = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM computer_data WHERE mac_address = ?)", macAddress).Scan(&dataExists)
	if err != nil {
		return fmt.Errorf("erro ao verificar existência de dados do computador: %v", err)
	}

	if dataExists {
		// Atualizar o registro existente
		_, err = tx.Exec(`
			UPDATE computer_data 
			SET data_json = ?, timestamp = ?
			WHERE mac_address = ?
		`, string(jsonData), now, macAddress)
		if err != nil {
			return fmt.Errorf("erro ao atualizar dados JSON: %v", err)
		}
	} else {
		// Inserir novo registro
		_, err = tx.Exec(`
			INSERT INTO computer_data (mac_address, data_json, timestamp)
			VALUES (?, ?, ?)
		`, macAddress, string(jsonData), now)
		if err != nil {
			return fmt.Errorf("erro ao salvar dados JSON: %v", err)
		}
	}

	// Commit da transação
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("erro ao finalizar transação: %v", err)
	}

	return nil
}

// Obtém todos os computadores do banco de dados
func getAllComputers() ([]map[string]interface{}, error) {
	rows, err := db.Query(`
		SELECT mac_address, hostname, ip_address, os_name, cpu_model, 
			   ram_total, disk_c_total, agent_version, last_seen, first_seen
		FROM computers
		ORDER BY hostname
	`)
	if err != nil {
		return nil, fmt.Errorf("erro ao consultar computadores: %v", err)
	}
	defer rows.Close()

	var computers []map[string]interface{}
	for rows.Next() {
		var mac, hostname, ip, os, cpu, agentVersion string
		var ram, disk float64
		var lastSeen, firstSeen time.Time

		err := rows.Scan(&mac, &hostname, &ip, &os, &cpu, &ram, &disk, &agentVersion, &lastSeen, &firstSeen)
		if err != nil {
			return nil, fmt.Errorf("erro ao ler dados do computador: %v", err)
		}

		computer := map[string]interface{}{
			"mac_address":   mac,
			"hostname":      hostname,
			"ip_address":    ip,
			"os_name":       os,
			"cpu_model":     cpu,
			"ram_total":     ram,
			"disk_c_total":  disk,
			"agent_version": agentVersion,
			"last_seen":     lastSeen.Format("2006-01-02 15:04:05"),
			"first_seen":    firstSeen.Format("2006-01-02 15:04:05"),
		}

		computers = append(computers, computer)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erro ao iterar resultados: %v", err)
	}

	return computers, nil
}

// Fecha a conexão com o banco de dados
func closeDatabase() {
	if db != nil {
		db.Close()
	}
}
