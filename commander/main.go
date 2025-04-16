package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)

func main() {
	// Configurar flags de linha de comando
	agentIP := flag.String("agent", "", "IP do agente para atualizar (ex: 192.168.1.100:9999 or 192.168.1.100)")
	updateIP := flag.String("update-ip", "", "Novo IP do servidor de atualização (ex: http://10.0.0.1:9991 or 10.0.0.1:9991)")
	getInfo := flag.String("info", "", "Obter informações detalhadas do agente. Opções: tudo, cpu, discos, gpu, hardware, memoria, processos, rede, sistema, agente")
	timeout := flag.Int("timeout", 5, "Timeout em segundos para requisições")
	sysInfoInterval := flag.Int("sys-interval", 0, "Atualizar intervalo de coleta de informações do sistema (em minutos)")
	updateInterval := flag.Int("update-interval", 0, "Atualizar intervalo de verificação de atualizações (em minutos)")
	flag.Parse()

	// Carregar a chave privada
	var err error
	privateKey, err = loadPrivateKey("keys/private_key.pem")
	if err != nil {
		log.Fatalf("Erro: Não foi possível carregar a chave privada: %v", err)
	} else {
		log.Println("Chave privada carregada com sucesso")
	}

	// Verificar se é para atualizar um agente
	if *agentIP != "" && *updateIP != "" {
		if privateKey == nil {
			log.Fatalf("Erro: Chave privada necessária para atualizar agentes")
		}

		err := updateAgentServerIP(*agentIP, *updateIP)
		if err != nil {
			log.Fatalf("Erro ao atualizar agente: %v", err)
		}

		log.Printf("Agente %s atualizado com sucesso para usar o servidor %s", *agentIP, *updateIP)
		return
	}

	// Verificar se é para atualizar o intervalo de coleta de informações do sistema
	if *agentIP != "" && *sysInfoInterval > 0 {
		if privateKey == nil {
			log.Fatalf("Erro: Chave privada necessária para atualizar configurações")
		}

		err := updateSystemInfoInterval(*agentIP, *sysInfoInterval)
		if err != nil {
			log.Fatalf("Erro ao atualizar intervalo de coleta: %v", err)
		}

		log.Printf("Intervalo de coleta de informações do sistema atualizado para %d minutos no agente %s", *sysInfoInterval, *agentIP)
		return
	}

	// Verificar se é para atualizar o intervalo de verificação de atualizações
	if *agentIP != "" && *updateInterval > 0 {
		if privateKey == nil {
			log.Fatalf("Erro: Chave privada necessária para atualizar configurações")
		}

		err := updateCheckInterval(*agentIP, *updateInterval)
		if err != nil {
			log.Fatalf("Erro ao atualizar intervalo de verificação: %v", err)
		}

		log.Printf("Intervalo de verificação de atualizações atualizado para %d minutos no agente %s", *updateInterval, *agentIP)
		return
	}

	// Verificar se é para obter informações do agente
	if *agentIP != "" && *getInfo != "" {
		if privateKey == nil {
			log.Fatalf("Erro: Chave privada necessária para obter informações criptografadas")
		}

		// Determinar qual endpoint consultar
		endpoint := strings.ToLower(*getInfo)

		// Se for "tudo" ou vazio, consultar o endpoint principal
		if endpoint == "tudo" || endpoint == "" {
			log.Printf("Consultando todas as informações do agente em %s...", *agentIP)
			info, err := getAgentInfo(*agentIP, *timeout, "")
			if err != nil {
				log.Fatalf("Erro ao obter informações do agente: %v", err)
			}

			// Exibir o JSON formatado
			jsonData, err := json.MarshalIndent(info, "", "  ")
			if err != nil {
				log.Fatalf("Erro ao formatar JSON: %v", err)
			}
			fmt.Println(string(jsonData))
			return
		}

		// Verificar se o endpoint é válido
		validEndpoints := []string{"cpu", "discos", "gpu", "hardware", "memoria", "processos", "rede", "sistema", "agente"}
		isValid := false
		for _, valid := range validEndpoints {
			if endpoint == valid {
				isValid = true
				break
			}
		}

		if !isValid {
			log.Fatalf("Endpoint inválido: %s. Opções válidas: tudo, cpu, discos, gpu, hardware, memoria, processos, rede, sistema, agente", endpoint)
		}

		// Consultar o endpoint específico
		log.Printf("Consultando informações de %s do agente em %s...", endpoint, *agentIP)
		info, err := getAgentInfo(*agentIP, *timeout, endpoint)
		if err != nil {
			log.Fatalf("Erro ao obter informações de %s do agente: %v", endpoint, err)
		}

		// Exibir o JSON formatado
		jsonData, err := json.MarshalIndent(info, "", "  ")
		if err != nil {
			log.Fatalf("Erro ao formatar JSON: %v", err)
		}
		fmt.Println(string(jsonData))
		return
	}

	// Se não foram fornecidos parâmetros, exibir ajuda
	flag.Usage()
	os.Exit(1)
}
