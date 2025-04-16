package main

import (
	"fmt"
)

// Funções auxiliares para conversão de tipos
func parseUint64(s string) uint64 {
	var result uint64
	fmt.Sscanf(s, "%d", &result)
	return result
}