package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

func main() {
	// Verificar variables de entorno requeridas
	cloudflaredToken := os.Getenv("CLOUDFLARED_TOKEN")
	if cloudflaredToken == "" {
		log.Fatal("La variable de entorno CLOUDFLARED_TOKEN es requerida")
	}

	caddyConfig := os.Getenv("CADDY_CONFIG")
	if caddyConfig == "" {
		log.Fatal("La variable de entorno CADDY_CONFIG es requerida")
	}

	// Escribir la configuración de Caddy en un archivo
	err := os.WriteFile("/app/Caddyfile", []byte(caddyConfig), 0644)
	if err != nil {
		log.Fatalf("Error al escribir el archivo Caddyfile: %v", err)
	}

	// Crear canales para monitorear la finalización de los procesos
	caddyDone := make(chan error, 1)
	cloudflaredDone := make(chan error, 1)

	// Iniciar Caddy en segundo plano
	caddyCmd := exec.Command("caddy", "run", "--config", "/app/Caddyfile")

	caddyStdout, err := caddyCmd.StdoutPipe()
	if err != nil {
		log.Fatalf("Error al obtener StdoutPipe de Caddy: %v", err)
	}

	caddyStderr, err := caddyCmd.StderrPipe()
	if err != nil {
		log.Fatalf("Error al obtener StderrPipe de Caddy: %v", err)
	}

	if err := caddyCmd.Start(); err != nil {
		log.Fatalf("Error al iniciar Caddy: %v", err)
	}

	// Monitorear y prefijar la salida de Caddy
	go prefixOutput("caddy", caddyStdout)
	go prefixOutput("caddy", caddyStderr)

	// Monitorear Caddy
	go func() {
		caddyDone <- caddyCmd.Wait()
	}()

	// Iniciar Cloudflared en segundo plano
	cloudflaredCmd := exec.Command("cloudflared", "tunnel", "--no-autoupdate", "run", "--token", cloudflaredToken)

	cloudflaredStdout, err := cloudflaredCmd.StdoutPipe()
	if err != nil {
		log.Fatalf("Error al obtener StdoutPipe de Cloudflared: %v", err)
	}

	cloudflaredStderr, err := cloudflaredCmd.StderrPipe()
	if err != nil {
		log.Fatalf("Error al obtener StderrPipe de Cloudflared: %v", err)
	}

	if err := cloudflaredCmd.Start(); err != nil {
		log.Fatalf("Error al iniciar Cloudflared: %v", err)
	}

	// Monitorear y prefijar la salida de Cloudflared
	go prefixOutput("cloudflared", cloudflaredStdout)
	go prefixOutput("cloudflared", cloudflaredStderr)

	// Monitorear Cloudflared
	go func() {
		cloudflaredDone <- cloudflaredCmd.Wait()
	}()

	// Configurar canal para escuchar señales de interrupción (Ctrl+C)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)

	fmt.Println("Servicios iniciados. Presiona Ctrl+C para terminar.")

	// Esperar a que ocurra una señal o que un proceso termine
	select {
	case err := <-caddyDone:
		if err != nil {
			fmt.Printf("[caddy] terminó con error: %v\n", err)
		} else {
			fmt.Println("[caddy] terminó inesperadamente.")
		}
		// Terminar Cloudflared
		cloudflaredCmd.Process.Signal(syscall.SIGTERM)
		cloudflaredCmd.Wait()
		os.Exit(1)
	case err := <-cloudflaredDone:
		if err != nil {
			fmt.Printf("[cloudflared] terminó con error: %v\n", err)
		} else {
			fmt.Println("[cloudflared] terminó inesperadamente.")
		}
		// Terminar Caddy
		caddyCmd.Process.Signal(syscall.SIGTERM)
		caddyCmd.Wait()
		os.Exit(1)
	case sig := <-sigs:
		fmt.Printf("\nRecibida señal: %v. Terminando servicios...\n", sig)
		// Terminar ambos procesos
		caddyCmd.Process.Signal(syscall.SIGTERM)
		cloudflaredCmd.Process.Signal(syscall.SIGTERM)
		caddyCmd.Wait()
		cloudflaredCmd.Wait()
		fmt.Println("Servicios terminados.")
	}
}

// Función para leer y prefijar la salida de los procesos
func prefixOutput(prefix string, reader io.Reader) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		fmt.Printf("[%s] %s\n", prefix, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		fmt.Printf("Error al leer la salida de %s: %v\n", prefix, err)
	}
}
