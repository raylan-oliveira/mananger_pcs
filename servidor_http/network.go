package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Função para consultar um agente HTTP
func consultarAgente(ip string, port int, timeout time.Duration, retries int) (map[string]interface{}, error) {
	for attempt := 0; attempt <= retries; attempt++ {
		// Verificando se o host está online com um timeout menor
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", ip, port), timeout/2)
		if err != nil {
			if attempt < retries {
				time.Sleep(time.Second)
				continue
			}
			return nil, fmt.Errorf("host não está respondendo após %d tentativas", retries+1)
		}
		conn.Close()
		
		// Consultando informações via HTTP
		client := &http.Client{
			Timeout: timeout,
		}
		
		// Modificado para solicitar dados criptografados
		resp, err := client.Get(fmt.Sprintf("http://%s:%d?encrypt=true", ip, port))
		if err != nil {
			if attempt < retries {
				fmt.Printf("Erro de conexão com %s. Tentando novamente (%d/%d)...\n", ip, attempt+1, retries)
				time.Sleep(time.Second)
				continue
			}
			return nil, fmt.Errorf("erro de conexão: %v", err)
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			if attempt < retries {
				fmt.Printf("Resposta inválida de %s (código %d). Tentando novamente (%d/%d)...\n", 
					ip, resp.StatusCode, attempt+1, retries)
				time.Sleep(time.Second)
				continue
			}
			return nil, fmt.Errorf("resposta inválida: %s", resp.Status)
		}
		
		// Lendo o corpo da resposta
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			if attempt < retries {
				fmt.Printf("Erro ao ler resposta de %s: %v. Tentando novamente (%d/%d)...\n", 
					ip, err, attempt+1, retries)
				time.Sleep(time.Second)
				continue
			}
			return nil, fmt.Errorf("erro ao ler resposta: %v", err)
		}
		
		// Descriptografando a resposta
		info, err := decryptData(body)
		if err != nil {
			if attempt < retries {
				fmt.Printf("Erro ao descriptografar resposta de %s: %v. Tentando novamente (%d/%d)...\n", 
					ip, err, attempt+1, retries)
				time.Sleep(time.Second)
				continue
			}
			return nil, fmt.Errorf("erro ao descriptografar: %v", err)
		}
		
		return info, nil
	}
	
	return nil, fmt.Errorf("falha após %d tentativas", retries)
}

// Função para descobrir agentes na rede
func descobrirAgentes(rede string, port int, maxWorkers int) map[string]map[string]interface{} {
	fmt.Printf("Descobrindo agentes na rede %s...\n", rede)
	
	// Gerando lista de IPs da rede
	ips, err := gerarListaIPs(rede)
	if err != nil {
		fmt.Printf("Erro ao processar a rede %s: %v\n", rede, err)
		fmt.Println("Formato correto: 192.168.1.0/24")
		return make(map[string]map[string]interface{})
	}
	
	totalIPs := len(ips)
	fmt.Printf("Escaneando %d endereços IP...\n", totalIPs)
	
	// Aumentando o número de workers para melhorar a performance
	// Usando um valor mais alto para maxWorkers se não for especificado adequadamente
	if maxWorkers < 50 {
		maxWorkers = 100 // Aumentando para 100 conexões simultâneas
	}
	
	resultados := make(map[string]map[string]interface{})
	var mutex sync.Mutex
	var wg sync.WaitGroup
	
	// Criando um canal para distribuir os IPs para os workers
	ipChan := make(chan string, totalIPs)
	
	// Contador de progresso
	ipsProcessados := 0
	var countMutex sync.Mutex
	
	// Criando workers fixos em vez de uma goroutine por IP
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			// Cada worker processa IPs do canal até que esteja vazio
			for ip := range ipChan {
				// Usando timeout menor para acelerar a verificação
				info, err := consultarAgente(ip, port, 2*time.Second, 0)
				
				countMutex.Lock()
				ipsProcessados++
				
				// Mostrando progresso
				if ipsProcessados%10 == 0 || ipsProcessados == totalIPs {
					fmt.Printf("Progresso: %d/%d IPs (%.1f%%)\n", 
						ipsProcessados, totalIPs, float64(ipsProcessados)/float64(totalIPs)*100)
				}
				countMutex.Unlock()
				
				if err == nil && info != nil {
					fmt.Printf("Agente encontrado: %s\n", ip)
					mutex.Lock()
					resultados[ip] = info
					mutex.Unlock()
				}
			}
		}()
	}
	
	// Alimentando o canal com todos os IPs
	for _, ip := range ips {
		ipChan <- ip
	}
	close(ipChan)
	
	// Aguardando todos os workers terminarem
	wg.Wait()
	return resultados
}

// Função para gerar lista de IPs a partir de uma notação CIDR
func gerarListaIPs(cidr string) ([]string, error) {
	// Verificando se é um IP único
	if !strings.Contains(cidr, "/") {
		return []string{cidr}, nil
	}
	
	// Processando como CIDR
	ip, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}
	
	var ips []string
	for ip := ip.Mask(ipNet.Mask); ipNet.Contains(ip); incrementIP(ip) {
		// Pulando o endereço de rede e broadcast para redes pequenas
		if ip[len(ip)-1] == 0 || ip[len(ip)-1] == 255 {
			continue
		}
		ips = append(ips, ip.String())
	}
	
	return ips, nil
}

// Função auxiliar para incrementar um endereço IP
func incrementIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}