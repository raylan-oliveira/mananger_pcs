package main

import (
	"fmt"
	"log"
	"os/exec"
	"runtime"
	"strings"
)

func main() {
	// Test Windows version
	cmd := exec.Command("powershell", "-Command", "(Get-WmiObject -class Win32_OperatingSystem).Version")
	output, err := cmd.Output()
	if err != nil {
		log.Fatalf("Error executing command: %v", err)
	}
	fmt.Printf("Windows Version: %s\n", strings.TrimSpace(string(output)))

	// Test systeminfo command
	cmd = exec.Command("systeminfo", "/fo", "list")
	output, err = cmd.Output()
	if err != nil {
		log.Printf("Error executing systeminfo: %v", err)
	} else {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.Contains(line, "OS Version") {
				fmt.Printf("OS Version from systeminfo: %s\n", strings.TrimSpace(line))
				break
			}
		}
	}

	// Display runtime values
	fmt.Println("\n--- Runtime Values ---")
	fmt.Printf("GOOS: %s\n", runtime.GOOS)
	fmt.Printf("GOARCH: %s\n", runtime.GOARCH)
	fmt.Printf("NumCPU: %d\n", runtime.NumCPU())
	fmt.Printf("NumGoroutine: %d\n", runtime.NumGoroutine())
	fmt.Printf("Version: %s\n", runtime.Version())
	
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("Memory Allocated: %v MB\n", m.Alloc/1024/1024)
	fmt.Printf("Total Memory Allocated: %v MB\n", m.TotalAlloc/1024/1024)
	fmt.Printf("System Memory: %v MB\n", m.Sys/1024/1024)
	fmt.Printf("Number of GC: %v\n", m.NumGC)
	
	// Get OS name from registry
	cmd = exec.Command("powershell", "-Command", "Get-ItemProperty -Path 'HKLM:\\SOFTWARE\\Microsoft\\Windows NT\\CurrentVersion' -Name ProductName | Select-Object -ExpandProperty ProductName")
	output, err = cmd.Output()
	if err != nil {
		log.Printf("Error getting OS name: %v", err)
	} else {
		fmt.Printf("\nOS Name from Registry: %s\n", strings.TrimSpace(string(output)))
	}
}
