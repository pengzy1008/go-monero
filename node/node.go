package node

import (
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"strconv"
	"time"
)

// 虚拟的门罗币节点
type Node struct {
	my_port    uint32
	network_id []byte
	peer_id    uint64
	listener   net.Listener
}

func CreateNode(network_type string, listen_port uint32) Node {
	// 生成peer_id
	max := new(big.Int).Lsh(big.NewInt(1), 64)
	random_num, err := rand.Int(rand.Reader, max)
	if err != nil {
		fmt.Println("Error generating random peer_id: ", err)
		return Node{}
	}
	// 确定network_id
	var network_id []byte
	switch network_type {
	case "mainnet":
		network_id = network_id_mainnet
	case "testnet":
		network_id = network_id_testnet
	default:
		network_id = network_id_mainnet
	}
	node := Node{
		my_port:    listen_port,
		network_id: network_id,
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

	// 2. 使用协程循环接收连接请求
	go func() {
		for {
			conn, err := node.listener.Accept()
			if err != nil {
				fmt.Println("Error accepting connection:", err)
				continue
			}

			// 3. 并发处理连接
			go node.handleIncomingConnection(&conn)
		}
	}()

	// 3. 节点接收命令
	var command string
	for {
		time.Sleep(1e8)
		fmt.Println("+==================================+")
		fmt.Println("|                                  |")
		fmt.Println("|         Fake Monero Node         |")
		fmt.Println("|                                  |")
		// fmt.Println("|-----------====INFO====-----------|")
		// fmt.Printf("| my_port:      %5d           |\n", node.my_port)
		// fmt.Printf("| peer_id:      %5s...        |\n", string(node.peer_id))
		// fmt.Printf("| network_type: %5s...        |\n", string(node.network_id))
		fmt.Println("+============= Command ============+")
		fmt.Println("| 1. Connect to a target node.     |")
		fmt.Println("| 0. Exit                          |")
		fmt.Println("+==================================+")
		fmt.Print("Your Command: ")
		fmt.Scanln(&command)
		switch command {
		case "1": // 和目标节点发起连接
			fmt.Print("Target IP: ")
			var target_ip string
			fmt.Scanln(&target_ip)
			fmt.Print("Target Port: ")
			var target_port uint16
			fmt.Scanln(&target_port)
			node.setOutgoingConnection(target_ip, target_port)
		case "0":
			return
		default:
			fmt.Println("Unknown command, please try again.")
		}
		// node.setOutgoingConnection("100.116.72.96", 38080)
	}
}

func (node *Node) handleIncomingConnection(conn *net.Conn) {
	defer (*conn).Close()
	// fmt.Println("Accept incoming connection from " + (*conn).RemoteAddr().String())

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
		if msg.command == commandHandshake {
			if msg.expect_response {
				response_msg := node.CreateHandshakeResponse()
				data_to_send := append(response_msg.header_bytes, response_msg.payload_bytes...)
				_, err := (*conn).Write(data_to_send)
				if err != nil {
					log.Println("Error sending data:", err)
					return
				} else {
					fmt.Println("Handshake response sent!")
				}
			} else {
				fmt.Println("Receive Handshake reponse!")
			}
			continue
		}
		if msg.command == commandPingPong && msg.expect_response {
			response_msg := node.CreatePongResponse()
			data_to_send := append(response_msg.header_bytes, response_msg.payload_bytes...)
			_, err := (*conn).Write(data_to_send)
			if err != nil {
				log.Println("Error sending data:", err)
				return
			} else {
				fmt.Println("Pong response sent!")
			}
			continue
		}
	}
}

func (node *Node) setOutgoingConnection(target_ip string, target_port uint16) {
	// 建立tcp连接
	address := target_ip + ":" + strconv.Itoa(int(target_port))
	conn, err := net.Dial("tcp", address)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fail to connect to target %s: %v\n", address, err)
		return
	}
	defer conn.Close()
	// fmt.Printf("Successfully connect to target %s\n", address)
	// 发送握手请求
	request_msg := node.CreateHandshakeRequest()
	data_to_send := append(request_msg.header_bytes, request_msg.payload_bytes...)
	_, err = conn.Write(data_to_send)
	if err != nil {
		log.Println("Error sending data:", err)
		return
	} else {
		fmt.Println("Handshake request sent!")
	}
	// 循环接收对端的响应
	for {
		// 读消息的header数据 33字节
		msg := LevinProtocolMessage{
			payload: make(map[string]interface{}),
		}
		res := msg.readHeader(&conn)
		if !res {
			return
		}

		// 读消息的payload data，消息的反序列化
		res = msg.readPayload(&conn)
		if !res {
			return
		}

		// 处理Handshake响应消息
		if msg.command == commandHandshake && !msg.expect_response {
			return
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
	msg.writeHeader(commandHandshake, uint64(len(msg.payload_bytes)), true)
	return msg
}

func (node *Node) CreateHandshakeResponse() LevinProtocolMessage {
	msg := LevinProtocolMessage{}
	payload_map := msg.writeHandshakeResponsePayload(node.my_port, node.network_id, node.peer_id, nil)
	msg.writePayload(payload_map)
	msg.writeHeader(commandHandshake, uint64(len(msg.payload_bytes)), false)
	return msg
}

func (node *Node) CreatePingRequest() LevinProtocolMessage {
	msg := LevinProtocolMessage{}
	msg.writeHeader(commandPingPong, 0, true)
	return msg
}

func (node *Node) CreatePongResponse() LevinProtocolMessage {
	msg := LevinProtocolMessage{}
	payload_map := msg.writePongResponsePayload(node.peer_id)
	msg.writePayload(payload_map)
	msg.writeHeader(commandPingPong, uint64(len(msg.payload_bytes)), false)
	return msg
}
