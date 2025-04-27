package main

import (
	"log"
	"net/http"
)

// fileServerHandler é o manipulador personalizado para servir apenas os arquivos necessários
func fileServerHandler(fileServer http.Handler) http.HandlerFunc {
	// Lista de arquivos permitidos
	allowedFiles := map[string]bool{
		"/agente_http.exe": true,
		"/version.txt":     true,
		"/public_key.pem":  true,
	}

	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Verificar se o arquivo solicitado está na lista de permitidos
		if !allowedFiles[path] && path != "/" {
			// Arquivo não permitido, retornar 404
			http.NotFound(w, r)
			log.Printf("Acesso negado: %s de %s", path, r.RemoteAddr)
			return
		}

		// Adicionar cabeçalhos para evitar cache
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")

		// Verificar se é uma requisição para a chave pública
		if path == "/public_key.pem" {
			// Redirecionar para o arquivo real na pasta keys
			r2 := new(http.Request)
			*r2 = *r
			r2.URL.Path = "/keys/public_key.pem"

			// Definir o tipo MIME correto para arquivos PEM
			w.Header().Set("Content-Type", "application/x-pem-file")
			w.Header().Set("Content-Disposition", "attachment; filename=\"public_key.pem\"")

			log.Printf("Servindo chave pública para %s", r.RemoteAddr)
			fileServer.ServeHTTP(w, r2)
			return
		}

		// Registrar download de arquivos importantes
		if path == "/agente_http.exe" {
			log.Printf("Download do agente: %s", r.RemoteAddr)
		} else if path == "/version.txt" {
			log.Printf("Verificação de versão: %s", r.RemoteAddr)
		}

		// Servir o arquivo
		fileServer.ServeHTTP(w, r)
	}
}
