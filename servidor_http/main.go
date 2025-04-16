package main

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	// Configurações
	// Obtendo a rede dinamicamente
	rede, err := getLocalNetwork()
	if err != nil {
		fmt.Printf("Aviso: Não foi possível determinar a rede local: %v\n", err)
		rede = "192.168.1.0/24" // Usando rede padrão como fallback
		fmt.Printf("Usando rede padrão: %s\n", rede)
	}
	port := 9999
	intervaloConsulta := 300 * time.Second // 5 minutos
	maxWorkers := 25

	fmt.Println("=== Servidor de Monitoramento HTTP ===")
	fmt.Printf("Monitorando a rede: %s\n", rede)
	fmt.Printf("Intervalo de consulta: %s\n", intervaloConsulta)

	// Verificando se o diretório de chaves existe
	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("Erro ao obter diretório atual: %v\n", err)
		return
	}

	keysDir := filepath.Join(currentDir, "keys")
	if _, err := os.Stat(keysDir); os.IsNotExist(err) {
		fmt.Printf("ERRO: Diretório de chaves não encontrado: %s\n", keysDir)
		fmt.Println("Execute primeiro o script generate_keys.go")
		return
	}

	// Verificando se a chave privada existe
	privateKeyPath := filepath.Join(keysDir, "private_key.pem")
	if _, err := os.Stat(privateKeyPath); os.IsNotExist(err) {
		fmt.Printf("ERRO: Chave privada não encontrada: %s\n", privateKeyPath)
		fmt.Println("Execute primeiro o script generate_keys.go")
		return
	}

	// Inicializar o banco de dados
	fmt.Println("Inicializando banco de dados...")
	if err := initDatabase(); err != nil {
		fmt.Printf("ERRO: Falha ao inicializar banco de dados: %v\n", err)
		return
	}
	defer closeDatabase()
	fmt.Println("Banco de dados inicializado com sucesso.")

	for {
		fmt.Println("\nIniciando descoberta de agentes...")
		inicio := time.Now()

		agentes := descobrirAgentes(rede, port, maxWorkers)

		fmt.Printf("\nAgentes encontrados: %d\n", len(agentes))
		for ip, info := range agentes {
			fmt.Printf("\nIP: %s\n", ip)

			// Salvar informações no banco de dados
			err := saveComputerInfo(info, ip)
			if err != nil {
				fmt.Printf("Erro ao salvar informações no banco de dados: %v\n", err)
			} else {
				fmt.Printf("Informações salvas no banco de dados com sucesso.\n")
			}

			// Extrair informações do sistema
			if sistema, ok := info["sistema"].(map[string]interface{}); ok {
				if nomeHost, ok := sistema["nome_host"]; ok {
					fmt.Printf("Nome do Host: %v\n", nomeHost)
				} else {
					fmt.Println("Nome do Host: N/A")
				}
			} else {
				fmt.Println("Nome do Host: N/A")
			}

			// Extrair informações da CPU
			if cpu, ok := info["cpu"].(map[string]interface{}); ok {
				if modelo, ok := cpu["modelo"]; ok {
					fmt.Printf("Processador: %v\n", modelo)
				} else {
					fmt.Println("Processador: N/A")
				}
			} else {
				fmt.Println("Processador: N/A")
			}

			// Extrair informações da memória
			if memoria, ok := info["memoria"].(map[string]interface{}); ok {
				if totalGB, ok := memoria["total_gb"]; ok {
					fmt.Printf("Memória RAM: %.2f GB\n", totalGB)
				} else {
					fmt.Println("Memória RAM: N/A")
				}
			} else {
				fmt.Println("Memória RAM: N/A")
			}

			// Extrair informações do sistema operacional
			if sistema, ok := info["sistema"].(map[string]interface{}); ok {
				if nomeSO, ok := sistema["nome_so"]; ok {
					fmt.Printf("Sistema Operacional: %v\n", nomeSO)
				} else {
					fmt.Println("Sistema Operacional: N/A")
				}
			} else {
				fmt.Println("Sistema Operacional: N/A")
			}

			// Extrair informações do disco principal (C:)
			if discos, ok := info["discos"].([]interface{}); ok && len(discos) > 0 {
				for _, d := range discos {
					disco, ok := d.(map[string]interface{})
					if !ok {
						continue
					}

					if dispositivo, ok := disco["dispositivo"]; ok && dispositivo == "C:" {
						if totalGB, ok := disco["total_gb"]; ok {
							fmt.Printf("Disco C: %.2f GB\n", totalGB)
						} else {
							fmt.Println("Disco C: N/A")
						}
						break
					}
				}
			} else {
				fmt.Println("Disco C: N/A")
			}

			// Extrair informações da versão do agente
			if agente, ok := info["agente"].(map[string]interface{}); ok {
				if versao, ok := agente["versao_agente"]; ok {
					fmt.Printf("Versão do Agente: %v\n", versao)
				} else {
					fmt.Println("Versão do Agente: N/A")
				}
			} else {
				fmt.Println("Versão do Agente: N/A")
			}

			// Extrair MAC da interface ativa
			mac, err := extractPrimaryMacAddress(info)
			if err == nil {
				fmt.Printf("MAC Address: %s\n", mac)
			} else {
				fmt.Println("MAC Address: N/A")
			}
		}

		// Exibir estatísticas do banco de dados
		computers, err := getAllComputers()
		if err != nil {
			fmt.Printf("Erro ao obter computadores do banco de dados: %v\n", err)
		} else {
			fmt.Printf("\nTotal de computadores no banco de dados: %d\n", len(computers))
		}

		tempoExecucao := time.Since(inicio)
		fmt.Printf("\nTempo de execução: %.2f segundos\n", tempoExecucao.Seconds())

		// Aguardando o próximo ciclo
		tempoEspera := intervaloConsulta - tempoExecucao
		if tempoEspera < time.Second {
			tempoEspera = time.Second
		}
		fmt.Printf("Próxima consulta em %.0f segundos...\n", tempoEspera.Seconds())
		time.Sleep(tempoEspera)
	}
}

// Função para obter a rede local baseada nas interfaces de rede
func getLocalNetwork() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, iface := range interfaces {
		// Ignorando interfaces loopback, desativadas ou virtuais
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		// Ignorando interfaces virtuais comuns
		if strings.Contains(strings.ToLower(iface.Name), "vmware") ||
			strings.Contains(strings.ToLower(iface.Name), "virtual") ||
			strings.Contains(strings.ToLower(iface.Name), "vbox") {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			// Verificando se é um endereço IP
			if ipnet, ok := addr.(*net.IPNet); ok {
				// Verificando se é IPv4 e não é loopback
				if ipnet.IP.To4() != nil && !ipnet.IP.IsLoopback() {
					ones, _ := ipnet.Mask.Size()

					// Calcular o endereço de rede (zerando os bits do host)
					ip := ipnet.IP.To4()
					mask := net.CIDRMask(ones, 32)
					network := ip.Mask(mask)

					// Retornar a rede com a máscara correta
					return fmt.Sprintf("%s/%d", network.String(), ones), nil
				}
			}
		}
	}

	return "", fmt.Errorf("nenhuma interface de rede adequada encontrada")
}
