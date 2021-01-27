package main

import (
	"fmt"
	"io"
	"net"
)

func main() {

	lis, err := net.Listen("tcp", "127.0.0.1:8000")

	if err != nil {
		panic(err)
	}

	for {
		conn, err := lis.Accept()
		defer conn.Close()

		if err != nil {
			panic(err)
		}

		go handleConn(conn)
	}
}

func handleConn(conn net.Conn) {

	for {
		buffer := make([]byte, 5)

		//使用 io.ReadFull 进行包读取
		recvNum, err := io.ReadFull(conn, buffer)

		if err == io.EOF {
			// client 连接关闭
			fmt.Println("client close")
			break
		}

		if err != nil {
			fmt.Println(err)
			break
		}

		msg := string(buffer[:recvNum])

		fmt.Println("recv from client:", msg)

		conn.Write([]byte("world : " + msg))
	}

}
