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

// statsHandler exibe estatísticas de clientes em uma página HTML
func statsHandler(w http.ResponseWriter, r *http.Request) {
	// Verificar se a requisição é local
	clientIP := getClientIP(r)
	if !isLocalRequest(clientIP) {
		http.Error(w, "Acesso negado", http.StatusForbidden)
		return
	}
	
	// Exibir estatísticas de clientes
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, "<html><head><title>Estatísticas do Servidor</title></head><body>")
	fmt.Fprintf(w, "<h1>Clientes Conectados</h1>")
	fmt.Fprintf(w, "<table border='1'><tr><th>IP</th><th>Requisições</th></tr>")
	
	clientsMutex.Lock()
	for ip, count := range activeClients {
		fmt.Fprintf(w, "<tr><td>%s</td><td>%d</td></tr>", ip, count)
	}
	clientsMutex.Unlock()
	
	fmt.Fprintf(w, "</table></body></html>")
}