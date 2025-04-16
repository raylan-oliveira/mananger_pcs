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

		size := float64(info.Size())
		unit := "B"
		
		if size > 1024*1024 {
			size = size / (1024 * 1024)
			unit = "MB"
		} else if size > 1024 {
			size = size / 1024
			unit = "KB"
		}

		log.Printf("Arquivo: %s (%.2f %s, modificado em %s)", 
			filename, size, unit, info.ModTime().Format("02/01/2006 15:04:05"))
	}
}