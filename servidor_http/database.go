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
			ram_total INTEGER,
			agent_version TEXT,
			servidor_atualizacao TEXT,
			system_info_update_interval INTEGER,
			update_check_interval INTEGER,
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
	var hostname, osName, cpuModel, agentVersion, servidorAtualizacao string
	var ramTotal float64
	var systemInfoUpdateInterval, updateCheckInterval int

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
		if total, ok := memoria["total"].(float64); ok {
			ramTotal = total
		}
	}

	// Extrair versão do agente e configurações adicionais
	if agente, ok := info["agente"].(map[string]interface{}); ok {
		if versao, ok := agente["versao_agente"].(string); ok {
			agentVersion = versao
		}

		// Extrair servidor de atualização
		if servidor, ok := agente["servidor_atualizacao"].(string); ok {
			servidorAtualizacao = servidor
		}

		// Extrair intervalos de atualização - corrigindo a extração para lidar com diferentes tipos numéricos
		if infoInterval, ok := agente["system_info_update_interval"]; ok {
			switch v := infoInterval.(type) {
			case float64:
				systemInfoUpdateInterval = int(v)
			case int:
				systemInfoUpdateInterval = v
			case int64:
				systemInfoUpdateInterval = int(v)
			case string:
				// Tentar converter string para int
				var intVal int
				if _, err := fmt.Sscanf(v, "%d", &intVal); err == nil {
					systemInfoUpdateInterval = intVal
				}
			}
		}

		if updateInterval, ok := agente["update_check_interval"]; ok {
			switch v := updateInterval.(type) {
			case float64:
				updateCheckInterval = int(v)
			case int:
				updateCheckInterval = v
			case int64:
				updateCheckInterval = int(v)
			case string:
				// Tentar converter string para int
				var intVal int
				if _, err := fmt.Sscanf(v, "%d", &intVal); err == nil {
					updateCheckInterval = intVal
				}
			}
		}

		// Adicionar log para debug
		fmt.Printf("Valores extraídos: servidor_atualizacao=%s, system_info_update_interval=%d, update_check_interval=%d\n",
			servidorAtualizacao, systemInfoUpdateInterval, updateCheckInterval)
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
				ram_total = ?, agent_version = ?, last_seen = ?,
				servidor_atualizacao = ?, system_info_update_interval = ?, update_check_interval = ?
			WHERE mac_address = ?
		`, hostname, ip, osName, cpuModel, ramTotal, agentVersion, now,
			servidorAtualizacao, systemInfoUpdateInterval, updateCheckInterval, macAddress)
		if err != nil {
			return fmt.Errorf("erro ao atualizar computador: %v", err)
		}
	} else {
		// Inserir novo registro
		_, err = tx.Exec(`
			INSERT INTO computers 
			(mac_address, hostname, ip_address, os_name, cpu_model, ram_total,  
			 agent_version, last_seen, first_seen, servidor_atualizacao, 
			 system_info_update_interval, update_check_interval)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, macAddress, hostname, ip, osName, cpuModel, ramTotal,
			agentVersion, now, now, servidorAtualizacao, systemInfoUpdateInterval, updateCheckInterval)
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
			   ram_total, agent_version, last_seen, first_seen,
			   servidor_atualizacao, system_info_update_interval, update_check_interval
		FROM computers
		ORDER BY hostname
	`)
	if err != nil {
		return nil, fmt.Errorf("erro ao consultar computadores: %v", err)
	}
	defer rows.Close()

	var computers []map[string]interface{}
	for rows.Next() {
		var mac, hostname, ip, os, cpu, agentVersion, servidorAtualizacao string
		var ramTotal float64
		var lastSeen, firstSeen time.Time
		var systemInfoUpdateInterval, updateCheckInterval int

		err := rows.Scan(&mac, &hostname, &ip, &os, &cpu, &ramTotal, &agentVersion,
			&lastSeen, &firstSeen, &servidorAtualizacao, &systemInfoUpdateInterval, &updateCheckInterval)
		if err != nil {
			return nil, fmt.Errorf("erro ao ler dados do computador: %v", err)
		}

		computer := map[string]interface{}{
			"mac_address":                 mac,
			"hostname":                    hostname,
			"ip_address":                  ip,
			"os_name":                     os,
			"cpu_model":                   cpu,
			"ram_total":                   ramTotal,
			"agent_version":               agentVersion,
			"last_seen":                   lastSeen.Format("2006-01-02 15:04:05"),
			"first_seen":                  firstSeen.Format("2006-01-02 15:04:05"),
			"servidor_atualizacao":        servidorAtualizacao,
			"system_info_update_interval": systemInfoUpdateInterval,
			"update_check_interval":       updateCheckInterval,
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
