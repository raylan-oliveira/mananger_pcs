package main

import (
	"net"
	"os/exec"
	"strings"
	"regexp"
)

func getNetworkInfo() map[string]interface{} {
	info := make(map[string]interface{})
	
	// Obter interfaces de rede
	interfaces, err := net.Interfaces()
	if err == nil {
		var networkInterfaces []map[string]interface{}
		
		// Obter informações detalhadas das interfaces usando PowerShell
		cmd := exec.Command("powershell", "-Command", "Get-NetAdapter | Select-Object Name, InterfaceDescription, Status, MacAddress, LinkSpeed | ConvertTo-Json")
		output, err := cmd.Output()
		
		// Mapa para armazenar informações detalhadas das interfaces
		detailedInfo := make(map[string]map[string]string)
		
		if err == nil {
			// Processar a saída JSON manualmente (simplificado)
			jsonStr := string(output)
			// Dividir por objetos
			re := regexp.MustCompile(`\{\s*"Name"\s*:\s*"([^"]+)"\s*,\s*"InterfaceDescription"\s*:\s*"([^"]*)"\s*,\s*"Status"\s*:\s*(\d+|"[^"]*")\s*,\s*"MacAddress"\s*:\s*"([^"]*)"\s*,\s*"LinkSpeed"\s*:\s*"([^"]*)"\s*\}`)
			matches := re.FindAllStringSubmatch(jsonStr, -1)
			
			for _, match := range matches {
				if len(match) >= 6 {
					name := match[1]
					detailedInfo[name] = map[string]string{
						"descricao": match[2],
						"status": match[3],
						"mac": match[4],
						"velocidade": match[5],
					}
				}
			}
		}
		
		for _, iface := range interfaces {
			// Ignorar interfaces de loopback e desativadas
			if iface.Flags&net.FlagLoopback != 0 {
				continue
			}
			
			netInterface := make(map[string]interface{})
			netInterface["nome"] = iface.Name
			netInterface["mac"] = iface.HardwareAddr.String()
			
			// Adicionar informações detalhadas se disponíveis
			if details, ok := detailedInfo[iface.Name]; ok {
				netInterface["descricao"] = details["descricao"]
				netInterface["status"] = details["status"]
				netInterface["velocidade"] = details["velocidade"]
			} else {
				// Valores padrão para evitar nil
				netInterface["descricao"] = "Não disponível"
				netInterface["status"] = "Desconhecido"
				netInterface["velocidade"] = "Desconhecido"
			}
			
			// Obter endereços IP
			addrs, err := iface.Addrs()
			if err == nil {
				var ipv4 []string
				var ipv6 []string
				
				for _, addr := range addrs {
					ipNet, ok := addr.(*net.IPNet)
					if !ok {
						continue
					}
					
					if ipNet.IP.To4() != nil {
						ipv4 = append(ipv4, ipNet.IP.String())
					} else {
						ipv6 = append(ipv6, ipNet.IP.String())
					}
				}
				
				netInterface["ipv4"] = ipv4
				netInterface["ipv6"] = ipv6
			}
			
			networkInterfaces = append(networkInterfaces, netInterface)
		}
		
		info["interfaces"] = networkInterfaces
	}
	
	// Obter informações de DNS
	cmd := exec.Command("powershell", "-Command", "Get-DnsClientServerAddress | Select-Object -ExpandProperty ServerAddresses")
	output, err := cmd.Output()
	if err == nil {
		lines := strings.Split(string(output), "\n")
		var dnsServers []string
		
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" {
				dnsServers = append(dnsServers, line)
			}
		}
		
		// Remover duplicatas
		uniqueDNS := make(map[string]bool)
		var uniqueDNSList []string
		
		for _, server := range dnsServers {
			if !uniqueDNS[server] {
				uniqueDNS[server] = true
				uniqueDNSList = append(uniqueDNSList, server)
			}
		}
		
		info["dns_servers"] = uniqueDNSList
	}
	
	return info
}