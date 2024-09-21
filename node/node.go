package node

import (
	"crypto/rand"
	"fmt"
	"gomonero/levin"
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
		network_id = levin.NetworkIdMainnet
	case "testnet":
		network_id = levin.NetworkIdTestnet
	default:
		network_id = levin.NetworkIdMainnet
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
		// 读消息
		msg := levin.LevinProtocolMessage{}
		res := msg.ReadBuffer(conn)
		if !res {
			return
		}

		// 处理消息
		if msg.GetCommand() == levin.CommandHandshake {
			if msg.GetExpectResponse() {
				response_msg := levin.LevinProtocolMessage{}
				response_msg.CreateHandshakeResponse(node.my_port, node.network_id, node.peer_id, nil)
				data_to_send := append(response_msg.HeaderBytes(), response_msg.PayloadBytes()...)
				_, err := (*conn).Write(data_to_send)
				if err != nil {
					log.Println("Error sending data:", err)
					return
				} else {
					fmt.Println("Handshake response sent!")
				}
			} else {
				fmt.Println("Receive Handshake response!")
			}
			continue
		}
		if msg.GetCommand() == levin.CommandPingPong {
			if msg.GetExpectResponse() {
				response_msg := levin.LevinProtocolMessage{}
				response_msg.CreatePongResponse(node.peer_id)
				data_to_send := append(response_msg.HeaderBytes(), response_msg.PayloadBytes()...)
				_, err := (*conn).Write(data_to_send)
				if err != nil {
					log.Println("Error sending data:", err)
					return
				} else {
					fmt.Println("Pong response sent!")
				}
			} else {
				fmt.Println("Receive Pong response")
			}
			continue
		}
		if msg.GetCommand() == levin.CommandTimedSync {
			if msg.GetExpectResponse() {
				response_msg := levin.LevinProtocolMessage{}
				response_msg.CreateTimedSyncResponse(node.my_port, node.network_id, node.peer_id, generateRamdomPeerlist(250))
				data_to_send := append(response_msg.HeaderBytes(), response_msg.PayloadBytes()...)
				_, err := (*conn).Write(data_to_send)
				if err != nil {
					log.Println("Error sending data:", err)
					return
				} else {
					fmt.Println("Timed Sync response sent!")
				}
			} else {
				fmt.Println("Receive Timed Sync response")
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
	request_msg := levin.LevinProtocolMessage{}
	request_msg.CreateHandshakeRequest(node.my_port, node.network_id, node.peer_id)
	data_to_send := append(request_msg.HeaderBytes(), request_msg.PayloadBytes()...)
	_, err = conn.Write(data_to_send)
	if err != nil {
		log.Println("Error sending data:", err)
		return
	} else {
		fmt.Println("Handshake request sent!")
	}
	// 循环接收对端的响应
	for {
		// 读消息
		msg := levin.LevinProtocolMessage{}
		res := msg.ReadBuffer(&conn)
		if !res {
			return
		}

		// 处理Handshake响应消息
		if msg.GetCommand() == levin.CommandHandshake && !msg.GetExpectResponse() {
			return
		}
	}
}

/*
=============

	Tools

=============
*/
func generateRamdomPeerlist(num_of_peer int) []levin.PeerlistEntry {
	if num_of_peer > 250 {
		num_of_peer = 250
	}
	peerlist := make([]levin.PeerlistEntry, num_of_peer)
	max := new(big.Int).Lsh(big.NewInt(1), 64)
	for i := 0; i < num_of_peer; i++ {
		random_num, _ := rand.Int(rand.Reader, max)
		peerlist[i].IP = uint32(random_num.Uint64())
		random_num, _ = rand.Int(rand.Reader, max)
		peerlist[i].Port = uint16(random_num.Uint64())
		random_num, _ = rand.Int(rand.Reader, max)
		peerlist[i].PeerId = random_num.Uint64()
	}
	return peerlist
}
