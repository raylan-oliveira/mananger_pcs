package main

// AgenteInfo representa as informações do agente
type AgenteInfo struct {
	VersaoAgente             string `json:"versao_agente"`
	ServidorAtualizacao      string `json:"servidor_atualizacao"`
	UpdateCheckInterval      string `json:"update_check_interval"`
	SystemInfoUpdateInterval string `json:"system_info_update_interval"`
}

// SystemInfo representa as informações do sistema
type SystemInfo struct {
	Sistema   map[string]interface{}   `json:"sistema"`
	CPU       map[string]interface{}   `json:"cpu"`
	Memoria   map[string]interface{}   `json:"memoria"`
	Discos    []map[string]interface{} `json:"discos"`
	Rede      map[string]interface{}   `json:"rede"`
	GPU       interface{}              `json:"gpu"`
	Processos map[string]interface{}   `json:"processos"`
	Hardware  interface{}              `json:"hardware"`
	Agente    AgenteInfo               `json:"agente"`
}
