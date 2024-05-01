package utils

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"github.com/libp2p/go-libp2p/core/host"
)


func WaitForSignal(ctx context.Context, h host.Host) {
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

    select {
    case <-sigCh:
        log.Println("Received shutdown signal, closing...")
    case <-ctx.Done():
        log.Println("Received disconnect signal, closing...")
    }

    if err := h.Close(); err != nil {
        log.Println("Error closing host:", err)
    }
}