package main

import (
	"fmt"
	"os"
	"os/exec"
)

// createStartupTask cria uma tarefa no Agendador de Tarefas do Windows para iniciar o aplicativo na inicialização
// Retorna (taskExists bool, error)
func createStartupTask() (bool, error) {
	// Obter o caminho do executável atual
	exePath, err := os.Executable()
	if err != nil {
		return false, fmt.Errorf("não foi possível obter o caminho do executável: %v", err)
	}

	// Nome da tarefa
	taskName := "AgenteHTTPStartup"

	// Verificar se a tarefa já existe
	checkCmd := exec.Command("schtasks", "/query", "/tn", taskName)
	_, err = checkCmd.CombinedOutput()

	// Se o comando não retornar erro, a tarefa já existe
	if err == nil {
		return true, nil
	}

	// Comando para criar a tarefa usando schtasks com configurações melhoradas
	cmd := exec.Command("schtasks", "/create", "/tn", taskName,
		"/tr", fmt.Sprintf("\"%s\"", exePath), // Usar diretamente o executável
		"/sc", "onstart", // Executar na inicialização
		"/ru", "SYSTEM", // Executar como SYSTEM
		"/rl", "HIGHEST", // Executar com privilégios elevados
		"/f",                // Forçar criação/substituição
		"/delay", "0001:00") // Atrasar 1 minuto após o boot

	// Executar o comando
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("erro ao criar tarefa agendada: %v - %s", err, string(output))
	}

	// Adicionar regra de firewall para permitir conexões na porta
	firewallCmd := exec.Command("netsh", "advfirewall", "firewall", "add", "rule",
		"name=AgenteHTTP",
		"dir=in",
		"action=allow",
		fmt.Sprintf("program=%s", exePath),
		"protocol=TCP",
		fmt.Sprintf("localport=%d", 9999))

	firewallOutput, err := firewallCmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Aviso: Não foi possível adicionar regra de firewall: %v - %s\n",
			err, string(firewallOutput))
	} else {
		fmt.Println("Regra de firewall adicionada com sucesso.")
	}

	return false, nil
}
