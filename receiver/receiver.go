package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/spf13/pflag"

	pb "github.com/cheggaaa/pb/v3"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	ma "github.com/multiformats/go-multiaddr"

	CONSTANT "ftp_server/src/constant"
	node "ftp_server/src/node"
	"ftp_server/src/stream"
	"ftp_server/src/transport"
	helper "ftp_server/src/utils"
)

type FileMetadata struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
}

type connectionNotifiee struct {
	mynode host.Host
	ctx    context.Context
	cancel context.CancelFunc
}

func (n *connectionNotifiee) Connected(net network.Network, conn network.Conn) {
	log.Printf("Connected to: %s", conn.RemotePeer().String())
}

func (n *connectionNotifiee) Disconnected(net network.Network, conn network.Conn) {
	log.Printf("Disconnected from: %s", conn.RemotePeer().String())
	log.Printf("Closing node receiver...")
	n.cancel()
}

func (n *connectionNotifiee) OpenedStream(network.Network, network.Stream) {}
func (n *connectionNotifiee) ClosedStream(network.Network, network.Stream) {}
func (n *connectionNotifiee) Listen(network.Network, ma.Multiaddr)         {}
func (n *connectionNotifiee) ListenClose(network.Network, ma.Multiaddr)    {}

var listenAddr = fmt.Sprintf("/ip4/%s/tcp/0", helper.GetOutboundIP().String())

var senderAddress string = "/ip4/172.31.27.187/tcp/53157/p2p/QmUoqMQsaGywsRfJ8C8B91vqZLP8hx5XGKkvWpd2AcLTrn"

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mynode, err := initNode()
	if err != nil {
		log.Println("Error initializing node:", err)
		return
	}
	defer mynode.Close()

	_, err = connectToSender(ctx, mynode)
	if err != nil {
		log.Println("Error connecting to sender:", err)
		return
	}

	setHandlers(mynode)

	helper.WaitForSignal(ctx, mynode)
}

func initNode() (host.Host, error) {
	pflag.StringVarP(&senderAddress, "senderAddress", "s", "", "Sender address to connect to")
	pflag.Parse()

	if senderAddress == "" {
		log.Println("No client address provided")
		return nil, errors.New("no client address provided")
	}

	mynode, err := node.InitNode(listenAddr)
	if err != nil {
		return nil, err
	}

	nodeAddrs, err := node.GetNodeAddrs(mynode)
	if err != nil {
		return nil, err
	}

	log.Printf("receiver is listening on %s", nodeAddrs[0])

	return mynode, nil
}

func connectToSender(ctx context.Context, mynode host.Host) (*peer.AddrInfo, error) {
	info, err := node.GetNodeAddrsFromMaddr(senderAddress)
	if err != nil {
		return nil, err
	}

	err = node.ConnectToNode(ctx, mynode, info)
	if err != nil {
		return nil, err
	}

	mynode.Network().Notify(&connectionNotifiee{mynode: mynode, ctx: ctx})

	return info, nil
}

func setHandlers(mynode host.Host) {
	setChatStreamHandler(mynode)
	setFileShareStreamHandler(mynode)
}

func setChatStreamHandler(mynode host.Host) {
	mynode.SetStreamHandler(protocol.ID(CONSTANT.ChatProtocolVersion), handleChatStream)
}

func handleChatStream(s network.Stream) {
	log.Printf("new chatstream with %s", s.Conn().RemotePeer().String())
	defer func() {
		stream.CloseStream(s)
		log.Printf("closed chatsteam with %s", s.Conn().RemotePeer().String())
	}()

	reader := bufio.NewReader(s)
	writer := bufio.NewWriter(s)

	for {
		message, err := transport.ReadMessage(reader)
		if err != nil {
			helper.HandleError(writer, err)
			return
		}
		log.Printf("Received message from client: %s", message)

		err = transport.SendMessage(writer, "CODE:400\n")
		if err != nil {
			helper.HandleError(writer, err)
			return
		}
	}
}

func setFileShareStreamHandler(mynode host.Host) {
	mynode.SetStreamHandler(protocol.ID(CONSTANT.FileShareProtocolVersion), handleFileShareStream)
}

func handleFileShareStream(s network.Stream) {
	log.Printf("New:> File share stream with %s", s.Conn().RemotePeer().String())

	defer func() {
		stream.CloseStream(s)
		log.Printf("Closed:> File share stream")
	}()
	writer := bufio.NewWriter(s)

	// confirmation := helper.GetUserInput("Want to receive file (y/n)? ")

	// writer.WriteString(confirmation + "\n")
	// err := writer.Flush()

	// if err != nil {
	// 	log.Printf("error flushing writer: %v", err)
	// }

	start := time.Now()
	fileReader := bufio.NewReader(s)
	log.Printf("Receiving file... \n")

	metadataJson, err := fileReader.ReadString('\n')
	if err != nil {
		log.Printf("error reading metadata: %v", err)
		return
	}
	metadataJson = strings.TrimSuffix(metadataJson, "\n")

	var metadata FileMetadata
	err = json.Unmarshal([]byte(metadataJson), &metadata)
	if err != nil {
		log.Printf("error parsing metadata: %v", err)
		return
	}
	log.Printf("File name: %s, size: %d bytes\n", metadata.Name, metadata.Size)

	dir := "./received_files/"
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			log.Printf("error creating directory: %v", err)
			return
		}
	}

	outFile, err := os.Create(dir + metadata.Name)
	if err != nil {
		log.Printf("error creating file: %v", err)
		return
	}
	defer outFile.Close()

	bar := pb.New64(metadata.Size)
	bar.Set(pb.Bytes, true) // Set the units to bytes
	bar.SetRefreshRate(time.Millisecond * 10)
	bar.Start()

	fileSize := metadata.Size

	buf := make([]byte, CONSTANT.CHUNKSIZE)

	for {
		n, err := fileReader.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Printf("error reading file: %v", err)
			return
		}
		fileSize = fileSize - int64(n)
		bar.Add(n)
		if fileSize == 0 {
			break
		}

		_, err = outFile.Write(buf[:n])
		if err != nil {
			log.Printf("error writing file: %v", err)
			return
		}

	}

	bar.Finish()
	end := time.Now()
	duration := end.Sub(start)
	log.Printf("Time taken to receive file:%v\n", duration)

	log.Println("File received successfully")

	// Send ACK after receiving file
	writer = bufio.NewWriter(s)
	_, err = writer.WriteString("ACK" + "\n")
	if err != nil {
		log.Printf("error writing ACK: %v", err)
		return
	}
	err = writer.Flush()
	if err != nil {
		log.Printf("error flushing writer: %v", err)
	}
}
