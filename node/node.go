package node

import (
	"bufio"
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"strconv"
)

// 虚拟的门罗币节点
type Node struct {
	my_port  uint16
	peer_id  uint64
	listener net.Listener
}

func CreateNode(listen_port uint16) Node {
	max := new(big.Int).Lsh(big.NewInt(1), 64)
	random_num, err := rand.Int(rand.Reader, max)
	if err != nil {
		fmt.Println("Error generating random peer_id: ", err)
		return Node{}
	}
	node := Node{
		my_port: listen_port,
		peer_id: random_num.Uint64(),
	}
	return node
}

func (node Node) Start() {
	// IO多路复用启动Server
	// 在Linux环境下，goroutine底层会调用epoll来实现高并发

	// 1. 创建监听器
	var err error
	node.listener, err = net.Listen("tcp", ":"+strconv.Itoa(int(node.my_port)))
	defer node.listener.Close()
	if err != nil {
		log.Println("Error create listener: ", err)
		os.Exit(1)
	}
	fmt.Println("Node Server is listening on port " + strconv.Itoa(int(node.my_port)))

	// 2. 循环接收连接请求
	for {
		conn, err := node.listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}

		// 3. 并发处理连接
		go handleConnectionRequest(conn)
	}
}

func handleConnectionRequest(conn net.Conn) {
	defer conn.Close()
	fmt.Println("Connected to " + conn.RemoteAddr().String())

	reader := bufio.NewReader(conn)
	for {
		// 读取客户端发送的数据
		message, err := reader.ReadString('\n')
		if err != nil {
			log.Println("Error reading from connection: ", err)
			return
		}
		fmt.Println("Receive message: " + message)
	}
}
