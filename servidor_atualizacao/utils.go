package main

import (
	"log"
	"os"
	"path/filepath"
)

// checkImportantFiles verifica e exibe informações sobre arquivos importantes
func checkImportantFiles(dir string) {
	// Lista de arquivos importantes para verificar
	importantFiles := []string{
		"agente_http.exe",
		"version.txt",
	}

	for _, filename := range importantFiles {
		path := filepath.Join(dir, filename)
		info, err := os.Stat(path)
		if err != nil {
			log.Printf("AVISO: Arquivo %s não encontrado", filename)
			continue
		}

		// Simplificar o cálculo de tamanho - usar KB para todos os arquivos
		sizeKB := float64(info.Size()) / 1024
		
		log.Printf("Arquivo: %s (%.1f KB, modificado em %s)", 
			filename, sizeKB, info.ModTime().Format("02/01/2006"))
	}
}