package transport

import (
	"bufio"
	"strings"
)

func ReadMessage(reader *bufio.Reader) (string, error) {
	message, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	message = strings.TrimSuffix(message, "\n")
	return message, nil
}