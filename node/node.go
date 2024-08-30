package node

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"strconv"
)

const LevinMessageHeaderLength = 33

var LevinSignature = []byte{0x01, 0x21, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01}

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
	fmt.Println("Accept incoming connection from " + conn.RemoteAddr().String())

	for {
		// 1. 创建一个大小为8字节的缓冲区，接收发送来的字节流数据的前8字节，前8字节为门罗币消息协议的levin signature
		buffer := make([]byte, 33)
		// 读取客户端发送的数据
		header_length, err := conn.Read(buffer)
		if err != nil {
			log.Println("Error reading levin message from connection: "+conn.RemoteAddr().String(), err)
			log.Println("Disconnect with connection " + conn.RemoteAddr().String())
			return
		}
		if header_length < LevinMessageHeaderLength {
			log.Printf("Error levin message header length. Expected length: %d, received data length: %d\n", LevinMessageHeaderLength, header_length)
			return
		}
		buffer_ptr := 0
		if !bytes.Equal(buffer[buffer_ptr:buffer_ptr+8], LevinSignature) {
			log.Println("Error receiving data from " + conn.RemoteAddr().String() + ": not Monero network protocol message.")
			return
		}
		buffer_ptr += 8
		fmt.Println("Great! Monero network protocol message!")

		// 2. 继续接收后8个字节的数据，表示消息的数据长度
		message_length := binary.LittleEndian.Uint64(buffer[buffer_ptr : buffer_ptr+8])
		buffer_ptr += 8
		fmt.Println("Monero network protocol message data length =", message_length)

		// 3. 接收1个字节的bool类型的reture_data数据，0表示不需要回复(request数据)，1表示需要回复(response数据)
		return_data := (buffer[buffer_ptr] != 0)
		buffer_ptr += 1
		if return_data {
			fmt.Println("This message is a request, it needs to be response.")
		} else {
			fmt.Println("This message is a response, there is no need to reponse.")
		}

		// 4. 接收4字节的Command数据，uint32类型，其值代表着消息的类型，比如1001是握手消息，1002是定时同步消息，1003是ping/pong消息等
		command := binary.LittleEndian.Uint32(buffer[buffer_ptr : buffer_ptr+4])
		buffer_ptr += 4
		fmt.Println("Message Command Type:", command)

		// 5. 接收4字节的Return Code参数 int32类型
		return_code := int32(binary.LittleEndian.Uint32(buffer[buffer_ptr : buffer_ptr+4]))
		buffer_ptr += 4
		fmt.Println("Return Code:", return_code)

		// 6. 接收4字节的uint32类型Flags数据
		flags := binary.LittleEndian.Uint32(buffer[buffer_ptr : buffer_ptr+4])
		buffer_ptr += 4
		fmt.Println("Flags:", flags)

		// 7. Version参数，uint32类型，固定为1
		version := binary.LittleEndian.Uint32(buffer[buffer_ptr : buffer_ptr+4])
		buffer_ptr += 4
		fmt.Println("Version:", version)

		if buffer_ptr == LevinMessageHeaderLength {
			fmt.Println("Header Parse complete!")
		}

		// n. 接收后续的数据
		buffer = make([]byte, 1024)
		remain_bytes_length, err := conn.Read(buffer)
		if err != nil {
			log.Println("Error reading remaining levin message from connection:"+conn.RemoteAddr().String(), err)
		}
		fmt.Printf("Remain data: %d bytes\n", remain_bytes_length)
		fmt.Printf("%x\n", buffer[:remain_bytes_length])
	}
}
