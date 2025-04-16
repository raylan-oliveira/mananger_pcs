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

	for {
		fmt.Println("\nIniciando descoberta de agentes...")
		inicio := time.Now()

		agentes := descobrirAgentes(rede, port, maxWorkers)

		fmt.Printf("\nAgentes encontrados: %d\n", len(agentes))
		for ip, info := range agentes {
			fmt.Printf("\nIP: %s\n", ip)

			if nomeHost, ok := info["nome_host"]; ok {
				fmt.Printf("Nome do Host: %v\n", nomeHost)
			} else {
				fmt.Println("Nome do Host: N/A")
			}

			if processador, ok := info["processador"]; ok {
				fmt.Printf("Processador: %v\n", processador)
			} else {
				fmt.Println("Processador: N/A")
			}

			if memoriaRAM, ok := info["memoria_ram"]; ok {
				fmt.Printf("Memória RAM: %v\n", memoriaRAM)
			} else {
				fmt.Println("Memória RAM: N/A")
			}
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
