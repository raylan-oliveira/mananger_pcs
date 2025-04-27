package main

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
)

// getLocalIPv4 retorna o primeiro endereço IPv4 não-loopback do host
func getLocalIPv4() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, iface := range interfaces {
		// Verificar se a interface está ativa (up) e não é loopback
		if iface.Flags&net.FlagUp != 0 && iface.Flags&net.FlagLoopback == 0 {
			addrs, err := iface.Addrs()
			if err != nil {
				continue
			}

			for _, addr := range addrs {
				if ipnet, ok := addr.(*net.IPNet); ok {
					if ip4 := ipnet.IP.To4(); ip4 != nil {
						return ip4.String(), nil
					}
				}
			}
		}
	}

	return "", fmt.Errorf("nenhum endereço IPv4 ativo encontrado")
}

// checkInternetConnectivity verifica se há conexão com a internet usando ping para gateway e DNS externo
func checkInternetConnectivity() bool {
	// Primeiro, tenta obter o gateway padrão
	gateway, err := getDefaultGateway()
	if err != nil {
		fmt.Printf("Aviso: Não foi possível determinar o gateway padrão: %v\n", err)
		return false
	}

	fmt.Printf("Verificando conectividade com o gateway (%s)...\n", gateway)

	// Tenta fazer ping para o gateway
	gatewayCmd := exec.Command("ping", "-n", "1", "-w", "2000", gateway)
	gatewayErr := gatewayCmd.Run()

	if gatewayErr != nil {
		fmt.Printf("Aviso: Não foi possível conectar ao gateway: %v\n", gatewayErr)
		return false
	}

	fmt.Println("Conectividade com o gateway confirmada.")
	fmt.Println("Verificando conectividade com a internet (8.8.8.8)...")

	// Tenta fazer ping para o DNS do Google (8.8.8.8)
	internetCmd := exec.Command("ping", "-n", "1", "-w", "3000", "8.8.8.8")
	internetErr := internetCmd.Run()

	if internetErr != nil {
		fmt.Println("Aviso: Gateway acessível, mas sem conectividade com a internet.")
		return false
	}

	return true
}

// getDefaultGateway obtém o endereço IP do gateway padrão
func getDefaultGateway() (string, error) {
	// Executar o comando route print para obter a tabela de rotas
	cmd := exec.Command("route", "print", "0.0.0.0")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("erro ao executar 'route print': %v", err)
	}

	// Converter a saída para string
	routeTable := string(output)

	// Procurar pela linha que contém 0.0.0.0 (rota padrão)
	// Formato típico: 0.0.0.0 0.0.0.0 192.168.1.1 192.168.1.100 10
	lines := strings.Split(routeTable, "\n")
	for _, line := range lines {
		// Remover espaços extras e dividir por espaços
		line = strings.TrimSpace(line)
		fields := strings.Fields(line)

		// Verificar se a linha tem pelo menos 5 campos e começa com 0.0.0.0
		if len(fields) >= 5 && fields[0] == "0.0.0.0" && fields[1] == "0.0.0.0" {
			// O gateway é o terceiro campo
			gateway := fields[2]
			// Verificar se parece um IP válido
			if net.ParseIP(gateway) != nil {
				return gateway, nil
			}
		}
	}

	return "", fmt.Errorf("gateway padrão não encontrado na tabela de rotas")
}
