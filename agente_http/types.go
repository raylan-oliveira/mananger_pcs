package main

// SystemInfo representa as informações do sistema
type SystemInfo struct {
	Sistema         map[string]interface{}   `json:"sistema"`
	CPU             map[string]interface{}   `json:"cpu"`
	Memoria         map[string]interface{}   `json:"memoria"`
	Discos          []map[string]interface{} `json:"discos"`
	Rede            map[string]interface{}   `json:"rede"`
	GPU             interface{}              `json:"gpu"`
	Processos       map[string]interface{}   `json:"processos"`
	UsuariosLogados []string                 `json:"usuarios_logados"`
	Hardware        interface{}              `json:"hardware"`
	VersaoAgente    string                   `json:"versao_agente"`
}
