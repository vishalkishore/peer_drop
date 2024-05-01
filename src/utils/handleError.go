package utils

import (
	"bufio"
	"ftp_server/src/transport"
	"fmt"
)

func HandleError(writer *bufio.Writer, err error) {
	fmt.Println("Error:", err)
	transport.SendMessage(writer, "Error: "+err.Error()+"\n")
}