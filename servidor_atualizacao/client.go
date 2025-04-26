package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

// getClientIP obtém o endereço IP real do cliente, considerando proxies
func getClientIP(r *http.Request) string {
	// Tentar obter IP de cabeçalhos de proxy comuns
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		// X-Forwarded-For pode conter múltiplos IPs, pegar o primeiro
		ips := strings.Split(ip, ",")
		return strings.TrimSpace(ips[0])
	}

	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}

	// Obter IP da conexão direta
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr // Retornar como está se não puder separar
	}

	return ip
}

// registerClient registra uma conexão de cliente
func registerClient(ip string) {
	clientsMutex.Lock()
	defer clientsMutex.Unlock()

	activeClients[ip]++
}

// isLocalRequest verifica se a requisição é de localhost
func isLocalRequest(ip string) bool {
	return ip == "127.0.0.1" || ip == "::1" || ip == "localhost"
}

// showClientStats exibe estatísticas de clientes periodicamente
func showClientStats() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		clientsMutex.Lock()
		if len(activeClients) > 0 {
			log.Println("=== Clientes Conectados ===")
			for ip, count := range activeClients {
				log.Printf("IP: %s - Requisições: %d", ip, count)
			}
			log.Println("==========================")
		}
		clientsMutex.Unlock()
	}
}


// getLocalIPv4 obtém o endereço IPv4 local da máquina
func getLocalIPv4() (string, error) {
	// Obter todas as interfaces de rede
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", fmt.Errorf("erro ao obter interfaces de rede: %v", err)
	}

	// Procurar por uma interface adequada
	for _, iface := range interfaces {
		// Ignorar interfaces desativadas
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		// Ignorar interfaces loopback
		if iface.Flags&net.FlagLoopback != 0 {
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
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // Não é IPv4
			}

			return ip.String(), nil
		}
	}

	return "", fmt.Errorf("nenhum endereço IPv4 encontrado")
}
