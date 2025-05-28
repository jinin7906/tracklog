package tcp_client

import (
	"fmt"
	"net"
)

func SendLogToTCP(address, data string) error {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return fmt.Errorf("[tcp_client] connect err %s: %v", address, err)
	}
	defer conn.Close()

	_, err = conn.Write([]byte(data + "\n"))
	if err != nil {
		return fmt.Errorf("[tcp_client] data send err %s: %v", address, err)
	}
	return nil
}
