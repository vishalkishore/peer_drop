package stream

import(
	"log"
	"context"
	"time"
	"fmt"
	host "github.com/libp2p/go-libp2p/core/host"
	peer "github.com/libp2p/go-libp2p/core/peer"
	
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/protocol"
)


func OpenStream(ctx context.Context, h host.Host, info *peer.AddrInfo, protocolID string) (network.Stream, error) {
	return h.NewStream(ctx, info.ID, protocol.ID(protocolID))
}

func CloseStream(s network.Stream) {
	err := s.Close()
	if err != nil {
		log.Fatalf("Error closing stream: %s", err)
	}
}

func TryOpenStream(ctx context.Context, h host.Host, info *peer.AddrInfo, protocol string) (network.Stream, error) {
    var streamToOpened network.Stream
    var err error

    maxRetries := 3
    for i := 0; i < maxRetries; i++ {
		
        streamToOpened, err = OpenStream(ctx, h, info, protocol)
        if err == nil {
            break
        }
        log.Printf("Error opening stream: %v, retrying...", err)
        time.Sleep(2 * time.Second)
    }

    if err != nil {
        return nil, fmt.Errorf("error opening stream: %w", err)
    }

    return streamToOpened, nil
}