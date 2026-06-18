package threads

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/Pinggy-io/pinggy-go/pinggy"
)

// HostLocalFile starts a local file server via the official Pinggy Go SDK to provide a public URL for a local file.
// It returns the public URL, a cleanup function, and any error encountered.
func HostLocalFile(filePath string) (string, func(), error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	parentDir := filepath.Dir(absPath)
	fileName := filepath.Base(absPath)

	// 1. Initialize Pinggy Tunnel
	listener, err := pinggy.Connect(pinggy.HTTP)
	if err != nil {
		return "", nil, fmt.Errorf("failed to connect to pinggy: %w", err)
	}
	
	// 2. Extract Public URL
	// RemoteUrls is a method on the listener
	urls := listener.RemoteUrls()
	if len(urls) == 0 {
		listener.Close()
		return "", nil, fmt.Errorf("pinggy initiated but no remote URLs provided")
	}
	
	var publicURL string
	for _, u := range urls {
		if strings.HasPrefix(u, "https://") {
			publicURL = u
			break
		}
	}
	
	if publicURL == "" {
		publicURL = strings.Replace(urls[0], "http://", "https://", 1)
	}
	
	fullURL := publicURL + "/" + fileName

	// 3. Start static file server using the Pinggy listener
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir(parentDir)))
	srv := &http.Server{Handler: mux}

	go func() {
		if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Pinggy server error: %v\n", err)
		}
	}()

	// Give the tunnel a moment to stabilize across Pinggy's global network
	time.Sleep(2 * time.Second)

	cleanup := func() {
		fmt.Println("🧵 Cleaning up Pinggy SDK hosting...")
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
		_ = listener.Close()
	}

	return fullURL, cleanup, nil
}
