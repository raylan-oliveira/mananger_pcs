package main

import (
	"net"
	"net/http"
	"strings"
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

// isLocalRequest verifica se a requisição é de localhost
func isLocalRequest(ip string) bool {
	return ip == "127.0.0.1" || ip == "::1" || ip == "localhost"
}