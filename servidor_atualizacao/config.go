package main

import (
	"crypto/rsa"
	"sync"
	"time"
)

// Configurações do servidor
var (
	port          int
	readTimeout   time.Duration
	writeTimeout  time.Duration
	idleTimeout   time.Duration
	maxHeaderMB   int
	activeClients map[string]int // Mapa para rastrear clientes ativos por IP
	clientsMutex  sync.Mutex     // Mutex para acesso seguro ao mapa de clientes
	privateKey    *rsa.PrivateKey // Chave privada para assinatura
)