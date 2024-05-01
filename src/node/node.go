package node

import (
	"fmt"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
)

func InitNode(listenAddrs ...string) (host.Host, error) {
	// Generate a new private key
	priv, _, err := crypto.GenerateKeyPair(crypto.RSA, 2048)
	if err != nil {
		return nil, fmt.Errorf("error generating key pair: %w", err)
	}

	opts := []libp2p.Option{
		libp2p.Identity(priv),
		libp2p.Security(noise.ID, noise.New),
	}

	if len(listenAddrs) > 0 {
		opts = append(opts, libp2p.ListenAddrStrings(listenAddrs...))
	}

	host, err := libp2p.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("error creating libp2p host: %w", err)
	}

	return host, nil
}
