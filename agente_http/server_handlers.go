package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// Constante para controlar se os dados devem ser criptografados
const encriptado = true

// Handler para fornecer informações do sistema rapidamente (sem atualizações)
func quickSystemInfoHandlerDataBase(w http.ResponseWriter, r *http.Request) {
	// Obter informações diretamente do banco de dados
	info, err := getSystemInfoFromDB()
	if err != nil {
		fmt.Printf("Erro ao obter informações do banco de dados: %v\n", err)
		// Se falhar ao obter do banco, usar o cache em memória
		info = cachedSystemInfo
	}

	// Converter para JSON
	jsonData, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		http.Error(w, fmt.Sprintf("Erro ao serializar dados: %v", err), http.StatusInternalServerError)
		return
	}

	if encriptado {
		// Criptografar os dados
		exePath, err := os.Executable()
		if err != nil {
			errMsg := fmt.Sprintf("Erro ao obter caminho do executável: %v", err)
			fmt.Println(errMsg)
			http.Error(w, errMsg, http.StatusInternalServerError)
			return
		}

		exeDir := filepath.Dir(exePath)
		keysDir := filepath.Join(exeDir, "keys")
		publicKeyPath := filepath.Join(keysDir, "public_key.pem")

		if _, err := os.Stat(publicKeyPath); os.IsNotExist(err) {
			errMsg := fmt.Sprintf("Chave pública não encontrada em %s", publicKeyPath)
			fmt.Println(errMsg)
			http.Error(w, errMsg, http.StatusInternalServerError)
			return
		}

		// Criptografar os dados
		encryptedData, err := encryptWithPublicKey(jsonData)
		if err != nil {
			errMsg := fmt.Sprintf("Erro ao criptografar dados: %v", err)
			fmt.Println(errMsg)
			http.Error(w, errMsg, http.StatusInternalServerError)
			return
		}

		// Definir cabeçalhos e enviar resposta
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(encryptedData))
	} else {
		// Enviar JSON sem criptografia
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonData)
	}
}

// Handler para fornecer informações do sistema calculando
func systemInfoAllCalcHandler(w http.ResponseWriter, r *http.Request) {
	// Coletar informações do sistema em tempo real
	infoCompleta, err := collectAllInfoSyscall()
	if err != nil {
		http.Error(w, fmt.Sprintf("[systemInfoAllCalcHandler] Erro ao coletar informações do sistema: %v", err), http.StatusInternalServerError)
		return
	}

	// Salva as informações coletadas no banco de dados
	err = saveSystemInfoToDB(infoCompleta)
	if err != nil {
		fmt.Printf("[systemInfoAllCalcHandler] Erro ao salvar informações no banco de dados: %v\n", err)
	}

	// Converter para JSON
	jsonData, err := json.MarshalIndent(infoCompleta, "", "  ")
	if err != nil {
		http.Error(w, fmt.Sprintf("Erro ao serializar dados: %v", err), http.StatusInternalServerError)
		return
	}

	if encriptado {
		// Criptografar os dados
		exePath, err := os.Executable()
		if err != nil {
			errMsg := fmt.Sprintf("Erro ao obter caminho do executável: %v", err)
			fmt.Println(errMsg)
			http.Error(w, errMsg, http.StatusInternalServerError)
			return
		}

		exeDir := filepath.Dir(exePath)
		keysDir := filepath.Join(exeDir, "keys")
		publicKeyPath := filepath.Join(keysDir, "public_key.pem")

		if _, err := os.Stat(publicKeyPath); os.IsNotExist(err) {
			errMsg := fmt.Sprintf("Chave pública não encontrada em %s", publicKeyPath)
			fmt.Println(errMsg)
			http.Error(w, errMsg, http.StatusInternalServerError)
			return
		}

		// Criptografar os dados
		encryptedData, err := encryptWithPublicKey(jsonData)
		if err != nil {
			errMsg := fmt.Sprintf("Erro ao criptografar dados: %v", err)
			fmt.Println(errMsg)
			http.Error(w, errMsg, http.StatusInternalServerError)
			return
		}

		// Definir cabeçalhos e enviar resposta
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(encryptedData))
	} else {
		// Enviar JSON sem criptografia
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonData)
	}
}

// Handler para atualizar o IP do servidor de atualização
func updateServerIPHandler(w http.ResponseWriter, r *http.Request) {
	// Apenas aceitar requisições POST
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	// Ler o corpo da requisição
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Erro ao ler corpo da requisição", http.StatusBadRequest)
		return
	}

	// Estrutura para deserializar o JSON
	type UpdateRequest struct {
		IP string `json:"ip_servidor"`
	}

	// Verificar se o corpo está vazio
	if len(body) == 0 {
		http.Error(w, "Corpo da requisição vazio", http.StatusBadRequest)
		return
	}

	// Tentar descriptografar o corpo com a chave pública
	decryptedData, err := decryptWithPublicKey(string(body))
	if err != nil {
		fmt.Printf("Erro ao descriptografar dados: %v\n", err)
		http.Error(w, "Erro ao descriptografar dados", http.StatusBadRequest)
		return
	}

	// Deserializar o JSON
	var request UpdateRequest
	err = json.Unmarshal([]byte(decryptedData), &request)
	if err != nil {
		http.Error(w, "Erro ao deserializar JSON", http.StatusBadRequest)
		return
	}

	// Verificar se o IP foi fornecido
	if request.IP == "" {
		http.Error(w, "IP do servidor não fornecido", http.StatusBadRequest)
		return
	}

	// Atualizar o IP no banco de dados
	err = updateServerIP(request.IP)
	if err != nil {
		fmt.Printf("Erro ao atualizar IP do servidor: %v\n", err)
		http.Error(w, "Erro ao atualizar IP do servidor", http.StatusInternalServerError)
		return
	}

	// Atualizar a variável global
	updateServerURL = request.IP

	// Responder com sucesso
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "IP do servidor de atualização alterado para: %s", request.IP)
	fmt.Printf("IP do servidor de atualização alterado para: %s\n", request.IP)
}

// Handler para atualizar o intervalo de atualização das informações do sistema
func updateSystemInfoIntervalHandler(w http.ResponseWriter, r *http.Request) {
	// Apenas aceitar requisições POST
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	// Ler o corpo da requisição
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Erro ao ler corpo da requisição", http.StatusBadRequest)
		return
	}

	// Estrutura para deserializar o JSON
	type UpdateRequest struct {
		Intervalo int    `json:"intervalo"`
		Senha     string `json:"senha"`
	}

	// Verificar se o corpo está vazio
	if len(body) == 0 {
		http.Error(w, "Corpo da requisição vazio", http.StatusBadRequest)
		return
	}

	// Tentar descriptografar o corpo com a chave pública
	decryptedData, err := decryptWithPublicKey(string(body))
	if err != nil {
		fmt.Printf("Erro ao descriptografar dados: %v\n", err)
		http.Error(w, "Erro ao descriptografar dados", http.StatusBadRequest)
		return
	}

	// Deserializar o JSON
	var request UpdateRequest
	err = json.Unmarshal([]byte(decryptedData), &request)
	if err != nil {
		http.Error(w, "Erro ao deserializar JSON", http.StatusBadRequest)
		return
	}

	// Verificar se o intervalo e a senha foram fornecidos
	if request.Intervalo <= 0 {
		http.Error(w, "Intervalo inválido", http.StatusBadRequest)
		return
	}

	if request.Senha == "" {
		http.Error(w, "Senha não fornecida", http.StatusBadRequest)
		return
	}

	// Atualizar o intervalo no banco de dados
	err = updateSystemInfoInterval(request.Intervalo)
	if err != nil {
		fmt.Printf("Erro ao atualizar intervalo de atualização: %v\n", err)
		http.Error(w, "Erro ao atualizar intervalo de atualização", http.StatusInternalServerError)
		return
	}

	// Atualizar a variável global e sinalizar a mudança
	systemInfoUpdateIntervalMinutes = request.Intervalo
	select {
	case systemInfoIntervalChanged <- true:
		// Sinal enviado com sucesso
	default:
		// Canal cheio, não bloquear
	}

	// Responder com sucesso
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Intervalo de atualização de informações alterado para: %d minutos", request.Intervalo)
	fmt.Printf("Intervalo de atualização de informações alterado para: %d minutos\n", request.Intervalo)
}

// Handler para atualizar o intervalo de verificação de atualizações
func updateCheckIntervalHandler(w http.ResponseWriter, r *http.Request) {
	// Apenas aceitar requisições POST
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	// Ler o corpo da requisição
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Erro ao ler corpo da requisição", http.StatusBadRequest)
		return
	}

	// Estrutura para deserializar o JSON
	type UpdateRequest struct {
		Intervalo int `json:"intervalo"`
	}

	// Verificar se o corpo está vazio
	if len(body) == 0 {
		http.Error(w, "Corpo da requisição vazio", http.StatusBadRequest)
		return
	}

	// Tentar descriptografar o corpo com a chave pública
	decryptedData, err := decryptWithPublicKey(string(body))
	if err != nil {
		fmt.Printf("Erro ao descriptografar dados: %v\n", err)
		http.Error(w, "Erro ao descriptografar dados", http.StatusBadRequest)
		return
	}

	// Deserializar o JSON
	var request UpdateRequest
	err = json.Unmarshal([]byte(decryptedData), &request)
	if err != nil {
		http.Error(w, "Erro ao deserializar JSON", http.StatusBadRequest)
		return
	}

	// Verificar se o intervalo foi fornecido
	if request.Intervalo <= 0 {
		http.Error(w, "Intervalo inválido", http.StatusBadRequest)
		return
	}

	// Atualizar o intervalo no banco de dados
	err = updateCheckInterval(request.Intervalo)
	if err != nil {
		fmt.Printf("Erro ao atualizar intervalo de verificação de atualizações: %v\n", err)
		http.Error(w, "Erro ao atualizar intervalo de verificação de atualizações", http.StatusInternalServerError)
		return
	}

	// Atualizar a variável global e sinalizar a mudança
	updateCheckIntervalMinutes = request.Intervalo
	select {
	case updateCheckIntervalChanged <- true:
		// Sinal enviado com sucesso
	default:
		// Canal cheio, não bloquear
	}

	// Responder com sucesso
	w.WriteHeader(http.StatusOK)
	fmt.Printf("Intervalo de verificação de atualizações alterado para: %d minutos\n", request.Intervalo)
}

// Handler para informações de CPU
func cpuHandler(w http.ResponseWriter, r *http.Request) {
	// Obter informações atualizadas de CPU
	cpuInfo := getCPUInfoSyscall()

	// Converter para JSON
	jsonData, err := json.MarshalIndent(cpuInfo, "", "  ")
	if err != nil {
		http.Error(w, fmt.Sprintf("Erro ao serializar dados: %v", err), http.StatusInternalServerError)
		return
	}

	// Verificar se deve criptografar os dados
	if encriptado {
		// Criptografar os dados
		encryptedData, err := encryptWithPublicKey(jsonData)
		if err != nil {
			errMsg := fmt.Sprintf("Erro ao criptografar dados: %v", err)
			fmt.Println(errMsg)
			http.Error(w, errMsg, http.StatusInternalServerError)
			return
		}

		// Definir cabeçalhos e enviar resposta
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(encryptedData))
	} else {
		// Enviar JSON sem criptografia
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonData)
	}
}

// Handler para informações de discos
func discosHandler(w http.ResponseWriter, r *http.Request) {
	// Obter informações atualizadas de discos
	discosInfo := getDiskInfoSyscall()

	// Criar um mapa para encapsular o array
	response := map[string]interface{}{
		"discos": discosInfo,
	}

	// Converter para JSON
	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		http.Error(w, fmt.Sprintf("Erro ao serializar dados: %v", err), http.StatusInternalServerError)
		return
	}

	// Verificar se deve criptografar os dados
	if encriptado {
		// Criptografar os dados
		encryptedData, err := encryptWithPublicKey(jsonData)
		if err != nil {
			errMsg := fmt.Sprintf("Erro ao criptografar dados: %v", err)
			fmt.Println(errMsg)
			http.Error(w, errMsg, http.StatusInternalServerError)
			return
		}

		// Definir cabeçalhos e enviar resposta
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(encryptedData))
	} else {
		// Enviar JSON sem criptografia
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonData)
	}
}

// Handler para informações de GPU
func gpuHandler(w http.ResponseWriter, r *http.Request) {
	// Obter informações atualizadas de GPU
	gpuInfo := getGPUInfoSyscall()

	// Criar um mapa para encapsular o array
	response := map[string]interface{}{
		"gpu": gpuInfo,
	}

	// Converter para JSON
	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		http.Error(w, fmt.Sprintf("Erro ao serializar dados: %v", err), http.StatusInternalServerError)
		return
	}

	// Verificar se deve criptografar os dados
	if encriptado {
		// Criptografar os dados
		encryptedData, err := encryptWithPublicKey(jsonData)
		if err != nil {
			errMsg := fmt.Sprintf("Erro ao criptografar dados: %v", err)
			fmt.Println(errMsg)
			http.Error(w, errMsg, http.StatusInternalServerError)
			return
		}

		// Definir cabeçalhos e enviar resposta
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(encryptedData))
	} else {
		// Enviar JSON sem criptografia
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonData)
	}
}

// Handler para informações de hardware
func hardwareHandler(w http.ResponseWriter, r *http.Request) {
	// Obter informações atualizadas de hardware
	hardwareInfo := getHardwareInfoSyscall()

	// Converter para JSON
	jsonData, err := json.MarshalIndent(hardwareInfo, "", "  ")
	if err != nil {
		http.Error(w, fmt.Sprintf("Erro ao serializar dados: %v", err), http.StatusInternalServerError)
		return
	}

	// Verificar se deve criptografar os dados
	if encriptado {
		// Criptografar os dados
		encryptedData, err := encryptWithPublicKey(jsonData)
		if err != nil {
			errMsg := fmt.Sprintf("Erro ao criptografar dados: %v", err)
			fmt.Println(errMsg)
			http.Error(w, errMsg, http.StatusInternalServerError)
			return
		}

		// Definir cabeçalhos e enviar resposta
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(encryptedData))
	} else {
		// Enviar JSON sem criptografia
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonData)
	}
}

// Handler para informações de memória
func memoriaHandler(w http.ResponseWriter, r *http.Request) {
	// Obter informações atualizadas de memória
	memoriaInfo := getMemoryInfoSyscall()

	// Converter para JSON
	jsonData, err := json.MarshalIndent(memoriaInfo, "", "  ")
	if err != nil {
		http.Error(w, fmt.Sprintf("Erro ao serializar dados: %v", err), http.StatusInternalServerError)
		return
	}

	// Verificar se deve criptografar os dados
	if encriptado {
		// Criptografar os dados
		encryptedData, err := encryptWithPublicKey(jsonData)
		if err != nil {
			errMsg := fmt.Sprintf("Erro ao criptografar dados: %v", err)
			fmt.Println(errMsg)
			http.Error(w, errMsg, http.StatusInternalServerError)
			return
		}

		// Definir cabeçalhos e enviar resposta
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(encryptedData))
	} else {
		// Enviar JSON sem criptografia
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonData)
	}
}

// Handler para informações de rede
func redeHandler(w http.ResponseWriter, r *http.Request) {
	// Obter informações atualizadas de rede
	redeInfo := getNetworkInfoSyscall()

	// Converter para JSON
	jsonData, err := json.MarshalIndent(redeInfo, "", "  ")
	if err != nil {
		http.Error(w, fmt.Sprintf("Erro ao serializar dados: %v", err), http.StatusInternalServerError)
		return
	}

	// Verificar se deve criptografar os dados
	if encriptado {
		// Criptografar os dados
		encryptedData, err := encryptWithPublicKey(jsonData)
		if err != nil {
			errMsg := fmt.Sprintf("Erro ao criptografar dados: %v", err)
			fmt.Println(errMsg)
			http.Error(w, errMsg, http.StatusInternalServerError)
			return
		}

		// Definir cabeçalhos e enviar resposta
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(encryptedData))
	} else {
		// Enviar JSON sem criptografia
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonData)
	}
}

// Handler para informações do sistema
func sistemaHandler(w http.ResponseWriter, r *http.Request) {
	// Obter informações do sistema em tempo real, sem usar cache
	sistemaInfo := getSystemInfoSyscall()

	// Converter para JSON
	jsonData, err := json.MarshalIndent(sistemaInfo, "", "  ")
	if err != nil {
		http.Error(w, fmt.Sprintf("Erro ao serializar dados: %v", err), http.StatusInternalServerError)
		return
	}

	if encriptado {
		// Criptografar os dados
		exePath, err := os.Executable()
		if err != nil {
			errMsg := fmt.Sprintf("Erro ao obter caminho do executável: %v", err)
			fmt.Println(errMsg)
			http.Error(w, errMsg, http.StatusInternalServerError)
			return
		}

		exeDir := filepath.Dir(exePath)
		keysDir := filepath.Join(exeDir, "keys")
		publicKeyPath := filepath.Join(keysDir, "public_key.pem")

		if _, err := os.Stat(publicKeyPath); os.IsNotExist(err) {
			errMsg := fmt.Sprintf("Chave pública não encontrada em %s", publicKeyPath)
			fmt.Println(errMsg)
			http.Error(w, errMsg, http.StatusInternalServerError)
			return
		}

		// Criptografar os dados
		encryptedData, err := encryptWithPublicKey(jsonData)
		if err != nil {
			errMsg := fmt.Sprintf("Erro ao criptografar dados: %v", err)
			fmt.Println(errMsg)
			http.Error(w, errMsg, http.StatusInternalServerError)
			return
		}

		// Definir cabeçalhos e enviar resposta
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(encryptedData))
	} else {
		// Enviar JSON sem criptografia
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonData)
	}
}

// Handler para informações do agente
func agenteHandler(w http.ResponseWriter, r *http.Request) {
	// Obter a versão atual do agente do banco de dados
	versaoAgente, err := getCurrentVersion()
	if err != nil {
		fmt.Printf("Erro ao obter versão do agente: %v\n", err)
		versaoAgente = "desconhecida"
	}

	// Obter o IP do servidor de atualização
	servidorAtualizacao, err := getUpdateServerIP()
	if err != nil {
		fmt.Printf("Erro ao obter servidor de atualização: %v\n", err)
		servidorAtualizacao = "desconhecido"
	}

	// Obter os intervalos de atualização
	systemInfoUpdateInterval, err := getSystemInfoUpdateInterval()
	if err != nil {
		fmt.Printf("Erro ao obter intervalo de atualização: %v\n", err)
		systemInfoUpdateInterval = 10
	}

	updateCheckInterval, err := getUpdateCheckInterval()
	if err != nil {
		fmt.Printf("Erro ao obter intervalo de verificação: %v\n", err)
		updateCheckInterval = 10
	}

	// Criar objeto AgenteInfo
	agenteInfo := AgenteInfo{
		VersaoAgente:             versaoAgente,
		ServidorAtualizacao:      servidorAtualizacao,
		SystemInfoUpdateInterval: fmt.Sprintf("%d", systemInfoUpdateInterval),
		UpdateCheckInterval:      fmt.Sprintf("%d", updateCheckInterval),
	}

	// Converter para JSON
	jsonData, err := json.MarshalIndent(agenteInfo, "", "  ")
	if err != nil {
		http.Error(w, fmt.Sprintf("Erro ao serializar dados: %v", err), http.StatusInternalServerError)
		return
	}

	// Verificar se deve criptografar os dados
	if encriptado {
		// Criptografar os dados
		encryptedData, err := encryptWithPublicKey(jsonData)
		if err != nil {
			errMsg := fmt.Sprintf("Erro ao criptografar dados: %v", err)
			fmt.Println(errMsg)
			http.Error(w, errMsg, http.StatusInternalServerError)
			return
		}

		// Definir cabeçalhos e enviar resposta
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(encryptedData))
	} else {
		// Enviar JSON sem criptografia
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonData)
	}
}

// Handler para fornecer informações do sistema via syscall
func syscallInfoHandler(w http.ResponseWriter, r *http.Request) {
	// Coletar informações do sistema usando syscalls diretos
	info := getAllSyscallInfo()

	// Converter para JSON
	jsonData, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		http.Error(w, fmt.Sprintf("Erro ao serializar dados: %v", err), http.StatusInternalServerError)
		return
	}

	// Verificar se a resposta deve ser criptografada
	encryptParam := r.URL.Query().Get("encrypt")
	shouldEncrypt := encriptado || encryptParam == "true"

	if shouldEncrypt {
		// Criptografar os dados
		exePath, err := os.Executable()
		if err != nil {
			errMsg := fmt.Sprintf("Erro ao obter caminho do executável: %v", err)
			fmt.Println(errMsg)
			http.Error(w, errMsg, http.StatusInternalServerError)
			return
		}

		exeDir := filepath.Dir(exePath)
		keysDir := filepath.Join(exeDir, "keys")
		publicKeyPath := filepath.Join(keysDir, "public_key.pem")

		if _, err := os.Stat(publicKeyPath); os.IsNotExist(err) {
			errMsg := fmt.Sprintf("Chave pública não encontrada em %s", publicKeyPath)
			fmt.Println(errMsg)
			http.Error(w, errMsg, http.StatusInternalServerError)
			return
		}

		// Criptografar os dados
		encryptedData, err := encryptWithPublicKey(jsonData)
		if err != nil {
			errMsg := fmt.Sprintf("Erro ao criptografar dados: %v", err)
			fmt.Println(errMsg)
			http.Error(w, errMsg, http.StatusInternalServerError)
			return
		}

		// Definir cabeçalhos e enviar resposta
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(encryptedData))
	} else {
		// Enviar JSON sem criptografia
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonData)
	}
}
