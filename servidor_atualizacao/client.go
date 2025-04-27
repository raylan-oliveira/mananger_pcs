package main

import (
	"fmt"
	"net"
)

// Removida a função showClientStats

// getLocalIPv4 obtém o endereço IPv4 local da máquina
func getLocalIPv4() (string, error) {
	// Tentar primeiro com uma conexão UDP (não estabelece conexão real)
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err == nil {
		defer conn.Close()
		localAddr := conn.LocalAddr().(*net.UDPAddr)
		return localAddr.IP.String(), nil
	}

	// Método alternativo se o primeiro falhar
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", fmt.Errorf("erro ao obter interfaces de rede: %v", err)
	}

	// Procurar por uma interface adequada
	for _, iface := range interfaces {
		// Ignorar interfaces desativadas ou loopback
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		// Obter endereços da interface
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		// Procurar por um endereço IPv4
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			// Verificar se é IPv4 e não é loopback
			if ip != nil && !ip.IsLoopback() && ip.To4() != nil {
				return ip.String(), nil
			}
		}
	}

	return "", fmt.Errorf("nenhum endereço IPv4 encontrado")
}
