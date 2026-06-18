package threads

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"time"
)

// HostLocalFile starts a local file server and a localhost.run tunnel to provide a public URL for a local file.
// It returns the public URL, a cleanup function, and any error encountered.
func HostLocalFile(filePath string) (string, func(), error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	parentDir := filepath.Dir(absPath)
	fileName := filepath.Base(absPath)

	// 1. Start local static file server on a random port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", nil, fmt.Errorf("failed to start local listener: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port

	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir(parentDir)))
	srv := &http.Server{Handler: mux}

	go func() {
		if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Local server error: %v\n", err)
		}
	}()

	// 2. Start localhost.run SSH Tunnel
	cmd := exec.Command("ssh", "-R", "80:localhost:"+strconv.Itoa(port), "nokey@localhost.run", "-T", "-o", "StrictHostKeyChecking=no")
	
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		listener.Close()
		return "", nil, fmt.Errorf("failed to get stdout pipe for localhost.run: %w", err)
	}
	cmd.Stderr = cmd.Stdout // Capture everything

	if err := cmd.Start(); err != nil {
		listener.Close()
		return "", nil, fmt.Errorf("failed to start localhost.run tunnel: %w", err)
	}

	// 3. Parse public URL from localhost.run output
	publicURL := ""
	urlRegex := regexp.MustCompile(`https://[a-zA-Z0-9.-]+\.lhr\.[a-z]+`)
	
	done := make(chan string)
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			match := urlRegex.FindString(line)
			if match != "" {
				done <- match
				return
			}
		}
		done <- ""
	}()

	select {
	case foundURL := <-done:
		if foundURL == "" {
			cmd.Process.Kill()
			srv.Shutdown(context.Background())
			return "", nil, fmt.Errorf("localhost.run stream closed or no URL found in output")
		}
		publicURL = foundURL
	case <-time.After(20 * time.Second):
		cmd.Process.Kill()
		srv.Shutdown(context.Background())
		return "", nil, fmt.Errorf("timeout waiting for localhost.run public URL")
	}

	// Give the tunnel a moment to stabilize
	time.Sleep(2 * time.Second)

	fullURL := publicURL + "/" + fileName
	
	cleanup := func() {
		fmt.Println("🧵 Cleaning up localhost.run hosting...")
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
		_ = listener.Close()
	}

	return fullURL, cleanup, nil
}
