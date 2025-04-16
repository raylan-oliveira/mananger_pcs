package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Função para descriptografar dados usando a chave privada
func decryptData(encryptedData []byte) (map[string]interface{}, error) {
	// Decodificando de base64
	encryptedBytes, err := base64.StdEncoding.DecodeString(string(encryptedData))
	if err != nil {
		return nil, fmt.Errorf("erro ao decodificar base64: %v", err)
	}

	// Carregando a chave privada
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("erro ao obter diretório atual: %v", err)
	}

	keysDir := filepath.Join(currentDir, "keys")
	privateKeyPath := filepath.Join(keysDir, "private_key.pem")

	privateKeyBytes, err := ioutil.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler arquivo de chave privada: %v", err)
	}

	block, _ := pem.Decode(privateKeyBytes)
	if block == nil {
		return nil, fmt.Errorf("falha ao decodificar chave privada PEM")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("falha ao analisar chave privada: %v", err)
	}

	// Separando os chunks criptografados
	var chunks [][]byte
	i := 0
	for i < len(encryptedBytes) {
		// Lendo o tamanho do chunk
		if i+4 >= len(encryptedBytes) {
			return nil, fmt.Errorf("formato inválido: tamanho do chunk não encontrado")
		}

		chunkLen := binary.BigEndian.Uint32(encryptedBytes[i : i+4])
		i += 4

		// Pulando o separador ':'
		if i >= len(encryptedBytes) || encryptedBytes[i] != ':' {
			return nil, fmt.Errorf("formato inválido: separador não encontrado")
		}
		i++

		// Lendo o chunk
		if i+int(chunkLen) > len(encryptedBytes) {
			return nil, fmt.Errorf("formato inválido: chunk incompleto")
		}

		chunk := encryptedBytes[i : i+int(chunkLen)]
		chunks = append(chunks, chunk)
		i += int(chunkLen)

		// Pulando o separador ':'
		if i >= len(encryptedBytes) || encryptedBytes[i] != ':' {
			return nil, fmt.Errorf("formato inválido: separador não encontrado após chunk")
		}
		i++
	}

	// Descriptografando cada chunk
	var decryptedData []byte
	for _, chunk := range chunks {
		decryptedChunk, err := rsa.DecryptOAEP(
			sha256.New(),
			rand.Reader,
			privateKey,
			chunk,
			nil,
		)
		if err != nil {
			return nil, fmt.Errorf("erro ao descriptografar chunk: %v", err)
		}
		decryptedData = append(decryptedData, decryptedChunk...)
	}

	// Convertendo para JSON
	var result map[string]interface{}
	err = json.Unmarshal(decryptedData, &result)
	if err != nil {
		return nil, fmt.Errorf("erro ao converter JSON: %v", err)
	}

	return result, nil
}

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

// Função para exibir informações detalhadas do sistema
func exibirInformacoesDetalhadas(info map[string]interface{}) {
	fmt.Println("\n=== Informações Detalhadas do Sistema ===")

	// Informações básicas
	fmt.Printf("Nome do Host: %v\n", info["nome_host"])
	fmt.Printf("Processador: %v\n", info["processador"])
	fmt.Printf("Memória RAM: %v\n", info["memoria_ram"])

	// Sistema
	if sistema, ok := info["sistema"].(map[string]interface{}); ok {
		fmt.Println("\n--- Sistema ---")
		fmt.Printf("Nome: %v\n", sistema["nome"])
		fmt.Printf("Arquitetura: %v\n", sistema["arquitetura"])
		fmt.Printf("Versão: %v\n", sistema["versao"])
		fmt.Printf("Data/Hora: %v\n", sistema["data_hora"])
		fmt.Printf("Uptime (minutos): %v\n", sistema["uptime"])
	}

	// CPU
	if cpu, ok := info["cpu"].(map[string]interface{}); ok {
		fmt.Println("\n--- CPU ---")
		fmt.Printf("Modelo: %v\n", cpu["modelo"])
		fmt.Printf("Núcleos: %v\n", cpu["nucleos"])
		// Removido: Núcleos Físicos e Núcleos Lógicos
	}

	// Memória
	if memoria, ok := info["memoria"].(map[string]interface{}); ok {
		fmt.Println("\n--- Memória ---")
		fmt.Printf("Total: %.2f GB\n", memoria["total_gb"])
		fmt.Printf("Disponível: %.2f GB\n", memoria["disponivel_gb"])
		fmt.Printf("Em Uso: %.2f GB\n", memoria["usada_gb"])
		fmt.Printf("Percentual de Uso: %.1f%%\n", memoria["percentual_uso"])
	}

	// Discos
	if discos, ok := info["discos"].([]interface{}); ok {
		fmt.Println("\n--- Discos ---")
		for i, d := range discos {
			disco, _ := d.(map[string]interface{})
			fmt.Printf("\nDisco %d:\n", i+1)
			fmt.Printf("  Dispositivo: %v\n", disco["dispositivo"])
			fmt.Printf("  Sistema de Arquivos: %v\n", disco["sistema_arquivos"])
			fmt.Printf("  Total: %.2f GB\n", disco["total_gb"])
			fmt.Printf("  Livre: %.2f GB\n", disco["livre_gb"])
			fmt.Printf("  Em Uso: %.2f GB\n", disco["usado_gb"])
			fmt.Printf("  Percentual de Uso: %.1f%%\n", disco["percentual_uso"])
		}
	}

	// Rede
	if rede, ok := info["rede"].(map[string]interface{}); ok {
		fmt.Println("\n--- Interfaces de Rede ---")
		if interfaces, ok := rede["interfaces"].([]interface{}); ok {
			for i, iface := range interfaces {
				netInterface, _ := iface.(map[string]interface{})
				fmt.Printf("\nInterface %d:\n", i+1)
				fmt.Printf("  Nome: %v\n", netInterface["nome"])

				// Usar valores padrão para evitar nil
				descricao := "Não disponível"
				if netInterface["descricao"] != nil {
					descricao = fmt.Sprintf("%v", netInterface["descricao"])
				}
				fmt.Printf("  Descrição: %s\n", descricao)

				status := "Desconhecido"
				if netInterface["status"] != nil {
					status = fmt.Sprintf("%v", netInterface["status"])
				}
				fmt.Printf("  Status: %s\n", status)

				fmt.Printf("  MAC: %v\n", netInterface["mac"])

				velocidade := "Desconhecido"
				if netInterface["velocidade"] != nil {
					velocidade = fmt.Sprintf("%v", netInterface["velocidade"])
				}
				fmt.Printf("  Velocidade: %s\n", velocidade)

				// Endereços IPv4
				if ipv4, ok := netInterface["ipv4"].([]interface{}); ok && len(ipv4) > 0 {
					fmt.Printf("  Endereços IPv4:\n")
					for _, ip := range ipv4 {
						fmt.Printf("    - %v\n", ip)
					}
				}
			}
		}

		// Exibir servidores DNS se disponíveis
		if dnsServers, ok := rede["dns_servers"].([]interface{}); ok && len(dnsServers) > 0 {
			fmt.Println("\nServidores DNS:")
			for _, dns := range dnsServers {
				fmt.Printf("  - %v\n", dns)
			}
		}
	}

	// GPU
	fmt.Println("\n--- GPU ---")
	if gpus, ok := info["gpu"].([]interface{}); ok {
		for i, g := range gpus {
			gpu, _ := g.(map[string]interface{})
			fmt.Printf("\nGPU %d:\n", i+1)
			fmt.Printf("  Nome: %v\n", gpu["nome"])
			fmt.Printf("  Memória: %.2f GB\n", gpu["memoria_gb"])
			fmt.Printf("  Versão do Driver: %v\n", gpu["versao_driver"])
		}
	} else {
		fmt.Printf("%v\n", info["gpu"])
	}

	// Processos
	if processos, ok := info["processos"].(map[string]interface{}); ok {
		fmt.Println("\n--- Processos ---")
		// Removed total process count display

		if topCPU, ok := processos["top_5_cpu"].([]interface{}); ok && len(topCPU) > 0 {
			fmt.Println("\nTop 5 Processos por CPU:")
			for i, p := range topCPU {
				proc, _ := p.(map[string]interface{})
				cpuValue := proc["cpu"]
				cpuStr := ""
				if cpuValue != nil && cpuValue != "" {
					cpuStr = fmt.Sprintf("%v", cpuValue)
				} else {
					cpuStr = "0"
				}
				fmt.Printf("  %d. %v (PID: %v) - CPU: %s, Memória: %.2f MB\n",
					i+1, proc["nome"], proc["pid"], cpuStr, proc["memoria_mb"])
			}
		}

		if topMem, ok := processos["top_5_memoria"].([]interface{}); ok && len(topMem) > 0 {
			fmt.Println("\nTop 5 Processos por Memória:")
			for i, p := range topMem {
				proc, _ := p.(map[string]interface{})
				cpuValue := proc["cpu"]
				cpuStr := ""
				if cpuValue != nil && cpuValue != "" {
					cpuStr = fmt.Sprintf("%v", cpuValue)
				} else {
					cpuStr = "0"
				}
				fmt.Printf("  %d. %v (PID: %v) - CPU: %s, Memória: %.2f MB\n",
					i+1, proc["nome"], proc["pid"], cpuStr, proc["memoria_mb"])
			}
		}
	}

	// Usuários Logados
	if usuarios, ok := info["usuarios_logados"].([]interface{}); ok {
		fmt.Println("\n--- Usuários Logados ---")
		for i, u := range usuarios {
			fmt.Printf("  %d. %v\n", i+1, u)
		}
	}

	// Hardware
	fmt.Println("\n--- Hardware ---")
	if hardware, ok := info["hardware"].(map[string]interface{}); ok {
		fmt.Printf("Fabricante: %v\n", hardware["fabricante"])
		fmt.Printf("Modelo: %v\n", hardware["modelo"])
		fmt.Printf("Número de Série: %v\n", hardware["numero_serie"])
		fmt.Printf("Versão BIOS: %v\n", hardware["versao_bios"])

		// Informações adicionais de hardware
		if hardware["tipo_sistema"] != nil {
			fmt.Printf("Tipo de Sistema: %v\n", hardware["tipo_sistema"])
		}
		if hardware["familia"] != nil {
			fmt.Printf("Família: %v\n", hardware["familia"])
		}
		if hardware["bios_fabricante"] != nil {
			fmt.Printf("Fabricante BIOS: %v\n", hardware["bios_fabricante"])
		}
		if hardware["bios_nome"] != nil {
			fmt.Printf("Nome BIOS: %v\n", hardware["bios_nome"])
		}
		if hardware["smbios_versao"] != nil {
			fmt.Printf("Versão SMBIOS: %v\n", hardware["smbios_versao"])
		}

		// Informações do registro do Windows sobre o BIOS
		if biosReg, ok := hardware["bios_registro"].(map[string]interface{}); ok && len(biosReg) > 0 {
			fmt.Println("\nInformações do Registro (BIOS):")
			// Ordenando as chaves para uma saída consistente
			var keys []string
			for k := range biosReg {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			for _, k := range keys {
				fmt.Printf("  %s: %v\n", k, biosReg[k])
			}
		}

		// Informações de placa-mãe
		if placaMae, ok := hardware["placa_mae"].(map[string]interface{}); ok {
			fmt.Println("\nPlaca-mãe:")
			if placaMae["fabricante"] != nil {
				fmt.Printf("  Fabricante: %v\n", placaMae["fabricante"])
			}
			if placaMae["modelo"] != nil {
				fmt.Printf("  Modelo: %v\n", placaMae["modelo"])
			}
			if placaMae["versao"] != nil {
				fmt.Printf("  Versão: %v\n", placaMae["versao"])
			}
			if placaMae["numero_serie"] != nil {
				fmt.Printf("  Número de Série: %v\n", placaMae["numero_serie"])
			}
		}

		// Uptime information
		if hardware["uptime_minutos"] != nil {
			uptime := hardware["uptime_minutos"]
			fmt.Printf("\nUptime (minutos): %v\n", uptime)

			// Se for um número, formatar em dias, horas e minutos
			if uptimeFloat, ok := uptime.(float64); ok {
				uptimeMinutes := int64(uptimeFloat)
				dias := uptimeMinutes / (24 * 60)
				horas := (uptimeMinutes % (24 * 60)) / 60
				minutos := uptimeMinutes % 60
				fmt.Printf("Uptime formatado: %d dias, %d horas, %d minutos\n", dias, horas, minutos)
			}
		}
	} else {
		fmt.Printf("%v\n", info["hardware"])
	}
}

func main() {
	// Verificando argumentos
	if len(os.Args) < 2 {
		fmt.Println("Uso: get_info <endereço_ip_ou_nome>")
		fmt.Println("Exemplos:")
		fmt.Println("  get_info 192.168.1.5")
		fmt.Println("  get_info FHRFILESERVER03")
		os.Exit(1)
	}

	// Obtendo o endereço do argumento
	address := os.Args[1]

	// Variável para armazenar o IP
	var ip string

	// Verificando se o endereço é um IP válido
	if net.ParseIP(address) != nil {
		// É um IP válido
		ip = address
	} else {
		// Não é um IP, tentando resolver como nome de host
		fmt.Printf("Resolvendo nome de host '%s'...\n", address)
		ips, err := net.LookupIP(address)
		if err != nil || len(ips) == 0 {
			fmt.Printf("Erro ao resolver nome de host '%s': %v\n", address, err)
			os.Exit(1)
		}

		// Usar o primeiro IP IPv4 encontrado
		for _, resolvedIP := range ips {
			// Verificar se é IPv4
			if ipv4 := resolvedIP.To4(); ipv4 != nil {
				ip = ipv4.String()
				fmt.Printf("Resolvido para IP: %s\n", ip)
				break
			}
		}

		// Se não encontrou IPv4, usar o primeiro IP da lista
		if ip == "" && len(ips) > 0 {
			ip = ips[0].String()
			fmt.Printf("Resolvido para IP: %s\n", ip)
		}

		// Se ainda não tiver IP, falhar
		if ip == "" {
			fmt.Printf("Não foi possível encontrar um endereço IPv4 para '%s'\n", address)
			os.Exit(1)
		}
	}

	// Configurações
	port := 9999
	timeout := 5 * time.Second
	retries := 2

	fmt.Printf("Consultando agente em %s:%d...\n", ip, port)

	// Consultando o agente
	inicio := time.Now()
	info, err := consultarAgente(ip, port, timeout, retries)
	if err != nil {
		fmt.Printf("Erro ao consultar agente: %v\n", err)
		os.Exit(1)
	}

	// Exibindo apenas o JSON formatado
	jsonData, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		fmt.Printf("Erro ao formatar JSON: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(jsonData))

	// Exibindo tempo de execução
	tempoExecucao := time.Since(inicio)
	fmt.Printf("\nTempo de execução: %.2f segundos\n", tempoExecucao.Seconds())
}
