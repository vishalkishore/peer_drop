package node

import (
	"context"
	"fmt"

	host "github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

func GetNodeAddrs(node host.Host) ([]ma.Multiaddr, error) {
	nodeInfo := peer.AddrInfo{
		ID:    node.ID(),
		Addrs: node.Addrs(),
	}
	return peer.AddrInfoToP2pAddrs(&nodeInfo)

}

func ConnectToNode(ctx context.Context, h host.Host, info *peer.AddrInfo) error {
	return h.Connect(ctx, *info)
}

func GetNodeAddrsFromMaddr(maddrStr string) (*peer.AddrInfo, error) {
	maddr, err := ma.NewMultiaddr(maddrStr)
	if err != nil {
		return nil, err
	}

	fmt.Printf("%s", maddr.String())

	return peer.AddrInfoFromP2pAddr(maddr)
}
