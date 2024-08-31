package node

import (
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
	my_port    uint32
	network_id []byte
	peer_id    uint64
	listener   net.Listener
}

func CreateNode(listen_port uint32) Node {
	max := new(big.Int).Lsh(big.NewInt(1), 64)
	random_num, err := rand.Int(rand.Reader, max)
	if err != nil {
		fmt.Println("Error generating random peer_id: ", err)
		return Node{}
	}
	node := Node{
		my_port:    listen_port,
		network_id: network_id_testnet,
		peer_id:    random_num.Uint64(),
	}
	return node
}

func (node *Node) Start() {
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
		go node.handleConnectionRequest(&conn)
	}
}

func (node *Node) handleConnectionRequest(conn *net.Conn) {
	defer (*conn).Close()
	fmt.Println("Accept incoming connection from " + (*conn).RemoteAddr().String())

	for {
		// 读消息的header数据 33字节
		msg := LevinProtocolMessage{
			payload: make(map[string]interface{}),
		}
		res := msg.readHeader(conn)
		if !res {
			return
		}

		// 读消息的payload data，消息的反序列化
		res = msg.readPayload(conn)
		if !res {
			return
		}

		// 处理消息
		if msg.command == 1001 && msg.expect_response {
			response_msg := node.CreateHandshakeResponse()
			data_to_send := append(response_msg.header_bytes, response_msg.payload_bytes...)
			_, err := (*conn).Write(data_to_send)
			if err != nil {
				log.Println("Error sending data:", err)
				return
			} else {
				fmt.Println("Handshake response sent!")
			}
		}
	}
}

/*
	==============================
	Create Monero Protocol Message
	==============================
*/

func (node *Node) CreateHandshakeRequest() LevinProtocolMessage {
	msg := LevinProtocolMessage{}
	payload_map := msg.writeHandshakeRequestPayload(node.my_port, node.network_id, node.peer_id)
	msg.writePayload(payload_map)
	msg.writeHeader(1001, uint64(len(msg.payload_bytes)), true)
	return msg
}

func (node *Node) CreateHandshakeResponse() LevinProtocolMessage {
	msg := LevinProtocolMessage{}
	payload_map := msg.writeHandshakeResponsePayload(node.my_port, node.network_id, node.peer_id, nil)
	msg.writePayload(payload_map)
	msg.writeHeader(1001, uint64(len(msg.payload_bytes)), false)
	return msg
}
