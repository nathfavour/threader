package threads

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
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
	p := pinggy.Connect()
	// Use default config (free tier, http tunnel)
	
	listener, err := p.Initiate()
	if err != nil {
		return "", nil, fmt.Errorf("failed to initiate pinggy tunnel: %w", err)
	}

	// 2. Extract Public URL
	// The SDK populates RemoteUrls after Initiation
	if len(p.RemoteUrls) == 0 {
		listener.Close()
		return "", nil, fmt.Errorf("pinggy initiated but no remote URLs provided")
	}
	
	publicURL := p.RemoteUrls[0]
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

	cleanup := func() {
		fmt.Println("🧵 Cleaning up Pinggy SDK hosting...")
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
		_ = listener.Close()
	}

	return fullURL, cleanup, nil
}
