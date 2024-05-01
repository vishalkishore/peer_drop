package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/atotto/clipboard"

	CONSTANT "ftp_server/src/constant"
	node "ftp_server/src/node"
	"ftp_server/src/stream"
	"ftp_server/src/transport"
	helper "ftp_server/src/utils"

	"github.com/libp2p/go-libp2p/core/network"
	ma "github.com/multiformats/go-multiaddr"
)

type connectionNotifiee struct {
	host   host.Host
	ctx    context.Context
	infoCh chan *peer.AddrInfo
}

func (n *connectionNotifiee) Connected(net network.Network, conn network.Conn) {
	// log.Printf("Connected to: %s", conn.RemotePeer().String())
	// log.Printf("Remote Multiaddr: %s", conn.RemoteMultiaddr().String())
	info := &peer.AddrInfo{
		ID:    conn.RemotePeer(),
		Addrs: []ma.Multiaddr{conn.RemoteMultiaddr()},
	}

	// Send info to the channel
	n.infoCh <- info
}

func (n *connectionNotifiee) Disconnected(net network.Network, conn network.Conn) {
	log.Printf("Disconnected from: %s", conn.RemotePeer().String())
}
func (n *connectionNotifiee) OpenedStream(network.Network, network.Stream) {}
func (n *connectionNotifiee) ClosedStream(network.Network, network.Stream) {}
func (n *connectionNotifiee) Listen(network.Network, ma.Multiaddr)         {}
func (n *connectionNotifiee) ListenClose(network.Network, ma.Multiaddr)    {}

var listenAddr = fmt.Sprintf("/ip4/%s/tcp/0", helper.GetOutboundIP().String())

func main() {
	ctx := context.Background()

	host, err := node.InitNode(listenAddr)

	if err != nil {
		log.Fatalf("Error creating sender: %s", err)
		return
	}
	defer host.Close()

	nodeAddrs, err := node.GetNodeAddrs(host)

	if err != nil {
		log.Fatalf("Error converting peer info to p2p addrs: %v", err)
	}

	log.Printf("sender is listening on %s", nodeAddrs[0])

	addrString := nodeAddrs[0].String()

	err = clipboard.WriteAll(addrString)
	if err != nil {
		log.Fatalf("Error copying to clipboard: %v", err)
	} else {
		log.Println("Node address copied to clipboard..")
	}

	infoCh := make(chan *peer.AddrInfo)

	log.Println("Waiting for connection...")
	host.Network().Notify(&connectionNotifiee{host: host, ctx: ctx, infoCh: infoCh})

	info := <-infoCh

	openSenderStreams(ctx, host, info)

	// helper.WaitForSignal(h)
}

func openSenderStreams(ctx context.Context, h host.Host, info *peer.AddrInfo) {

	for {
		fmt.Printf("Enter:\n1 for chat stream\n2 for file share stream:\n3 to exit:\n> ")
		var choice int
		_, err := fmt.Scanln(&choice)
		if err != nil {
			fmt.Println("Error reading input:", err)
			return
		}
		switch choice {
		case 1:
			err = handleChatStream(ctx, h, info)
		case 2:
			filename, err := helper.ReadUserInput("Enter the name of the file to send: ")
			if err != nil {
				log.Fatalf("Error reading file name: %v", err)
				return
			}
			err = handleFileShareStream(ctx, h, info, filename)
		case 3:
			fmt.Println("Exiting...")
			return
		default:
			fmt.Println("Invalid choice")
			return
		}

		if err != nil {
			fmt.Println("Error:", err)
		}
	}
}

// handleChatStream handles the chat stream.
func handleChatStream(ctx context.Context, h host.Host, info *peer.AddrInfo) error {

	chatStream, err := stream.TryOpenStream(ctx, h, info, CONSTANT.ChatProtocolVersion)
	if err != nil {
		return err
	}
	defer stream.CloseStream(chatStream)

	reader := bufio.NewReader(chatStream)
	writer := bufio.NewWriter(chatStream)

	for {
		message, err := helper.ReadUserInput("Enter a message to send to the server: ")

		if err != nil {
			return fmt.Errorf("error reading input: %w", err)
		}

		_, err = writer.WriteString(message + "\n")
		if err != nil {
			return fmt.Errorf("error writing to server: %w", err)
		}
		writer.Flush()

		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("error reading from server: %w", err)
		}
		log.Printf("Received response from server: %s\n", response)

		confirmation, err := helper.ReadUserInput("Do you want to continue (yes/no)? ")
		if err != nil {
			return fmt.Errorf("error reading input: %w", err)
		}
		if confirmation != "yes" {
			break
		}
	}

	return nil
}

// handleFileShareStream handles the file share stream.
func handleFileShareStream(ctx context.Context, h host.Host, info *peer.AddrInfo, filename string) error {

	fileShareStream, err := stream.TryOpenStream(ctx, h, info, CONSTANT.FileShareProtocolVersion)
	if err != nil {
		return err
	}
	defer stream.CloseStream(fileShareStream)

	writer := bufio.NewWriter(fileShareStream)

	file, err := os.Open(filename)
	if err != nil {
		helper.HandleError(writer, err)
		return err
	}
	defer file.Close()

	reader := bufio.NewReader(fileShareStream)
	// Send the file
	// conf, err := transport.ReadMessage(reader)
	// if err != nil {
	//     return fmt.Errorf("error reading acknowledgement: %w", err)
	// }
	// if conf != "y" {
	//     return fmt.Errorf("receiver refused to receive file")
	// }

	err = transport.SendFile(writer, file)
	if err != nil {
		helper.HandleError(writer, err)
		log.Fatalf("Error sending file: %v", err)
		return err
	}

	log.Println("File sent successfully")
	// Wait for acknowledgement from receiver

	log.Printf("Waiting for acknowledgement from receiver...\n")
	ack, err := transport.ReadMessage(reader)
	if err != nil {
		return fmt.Errorf("error reading acknowledgement: %w", err)
	}
	if ack != "ACK" {
		return fmt.Errorf("did not receive correct acknowledgement from receiver")
	}
	log.Println("File received successfully")

	return nil
}
