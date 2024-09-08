package node

import (
	"bytes"
	"encoding/binary"
	"log"
	"net"
)

// header
var levinMessageHeaderLength = 33
var levinSignature = []byte{0x01, 0x21, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01}
var levinPacketRequest = uint32(1)
var levinPacketResponse = uint32(2)
var levinProtocolVer1 = uint32(1)
var levinRequestReturnCode = int32(0)
var levinResponseReturnCode = int32(1)

// payload header
var network_id_mainnet = []byte{0x12, 0x30, 0xf1, 0x71, 0x61, 0x04, 0x41, 0x61, 0x17, 0x31, 0x00, 0x82, 0x16, 0xa1, 0xa1, 0x10}
var network_id_testnet = []byte{0x12, 0x30, 0xf1, 0x71, 0x61, 0x04, 0x41, 0x61, 0x17, 0x31, 0x00, 0x82, 0x16, 0xa1, 0xa1, 0x11}
var genesis_hash_mainnet = []byte{
	0x41, 0x80, 0x15, 0xbb, 0x9a, 0xe9, 0x82, 0xa1, 0x97, 0x5d, 0xa7, 0xd7, 0x92, 0x77, 0xc2, 0x70,
	0x57, 0x27, 0xa5, 0x68, 0x94, 0xba, 0x0f, 0xb2, 0x46, 0xad, 0xaa, 0xbb, 0x1f, 0x46, 0x32, 0xe3,
}
var genesis_hash_testnet = []byte{
	0x48, 0xca, 0x7c, 0xd3, 0xc8, 0xde, 0x5b, 0x6a, 0x4d, 0x53, 0xd2, 0x86, 0x1f, 0xbd, 0xae, 0xdc,
	0xa1, 0x41, 0x55, 0x35, 0x59, 0xf9, 0xbe, 0x95, 0x20, 0x06, 0x80, 0x53, 0xcd, 0xa8, 0x43, 0x0b,
}

var p2pSupportFlagFluffyBlocks = byte(0x01)
var p2pSupportFlags = p2pSupportFlagFluffyBlocks

var portableStorageSignature1 = []byte{0x01, 0x11, 0x01, 0x01}
var portableStorageSignature2 = []byte{0x01, 0x01, 0x02, 0x01}
var portableStorageFormatVer = byte(1)

// payload data
var portableRawSizeMarkMask = byte(3)
var portableRawSizeMarkByte = byte(0)
var portableRawSizeMarkWord = byte(1)
var portableRawSizeMarkDword = byte(2)
var portableRawSizeMarkInt64 = byte(3)

var serializeTypeInt64 = byte(1)
var serializeTypeInt32 = byte(2)
var serializeTypeInt16 = byte(3)
var serializeTypeInt8 = byte(4)
var serializeTypeUint64 = byte(5)
var serializeTypeUint32 = byte(6)
var serializeTypeUint16 = byte(7)
var serializeTypeUint8 = byte(8)
var serializeTypeString = byte(10)
var serializeTypeObject = byte(12)
var serializeTypeArray = byte(13)
var serializeFlagArray = byte(0x80)

// var serializeTypeDouble = byte(9)
// var serializeTypeBool = byte(11)

type LevinProtocolMessage struct {
	// 字节流数据的缓冲区，接收对端的数据，暂存要发送给对端的数据
	header_bytes  []byte
	payload_bytes []byte
	ptr           uint64 // 读指针

	// header的反序列化后的字段
	signature       uint64
	length          uint64
	expect_response bool
	command         uint32
	return_code     int32
	flags           uint32
	version         uint32

	// payload的反序列化后的字段
	payload map[string]interface{}
}

/*
===============

	反序列化

===============
*/

// 读取消息的头部（包含头部的解析、反序列化）
func (msg *LevinProtocolMessage) readHeader(conn *net.Conn) bool {
	msg.header_bytes = make([]byte, levinMessageHeaderLength)
	// 读取header
	header_length, err := (*conn).Read(msg.header_bytes)
	if err != nil {
		log.Println("Error reading levin message from connection: "+(*conn).RemoteAddr().String(), err)
		log.Println("Disconnect with connection " + (*conn).RemoteAddr().String())
		return false
	}
	if header_length < levinMessageHeaderLength {
		log.Printf("Error levin message header length. Expected length: %d, received data length: %d\n", levinMessageHeaderLength, header_length)
		return false
	}
	// 1. 读取前8个字节的signature，判断是不是门罗币网络层协议消息
	msg.ptr = 0
	if !bytes.Equal(msg.header_bytes[msg.ptr:msg.ptr+8], levinSignature) {
		log.Println("Error receiving data from " + (*conn).RemoteAddr().String() + ": not Monero network protocol message.")
		return false
	}
	msg.ptr += 8
	msg.signature = binary.LittleEndian.Uint64(levinSignature)

	// 2. 继续接收后8个字节的数据，表示消息的数据长度
	msg.length = binary.LittleEndian.Uint64(msg.header_bytes[msg.ptr : msg.ptr+8])
	msg.ptr += 8

	// 3. 接收1个字节的bool类型的reture_data数据，0表示不需要回复(request数据)，1表示需要回复(response数据)
	msg.expect_response = (msg.header_bytes[msg.ptr] != 0)
	msg.ptr += 1

	// 4. 接收4字节的Command数据，uint32类型，其值代表着消息的类型，比如1001是握手消息，1002是定时同步消息，1003是ping/pong消息等
	msg.command = binary.LittleEndian.Uint32(msg.header_bytes[msg.ptr : msg.ptr+4])
	msg.ptr += 4

	// 5. 接收4字节的Return Code参数 int32类型
	msg.return_code = int32(binary.LittleEndian.Uint32(msg.header_bytes[msg.ptr : msg.ptr+4]))
	msg.ptr += 4

	// 6. 接收4字节的uint32类型Flags数据
	msg.flags = binary.LittleEndian.Uint32(msg.header_bytes[msg.ptr : msg.ptr+4])
	msg.ptr += 4

	// 7. Version参数，uint32类型，固定为1
	msg.version = binary.LittleEndian.Uint32(msg.header_bytes[msg.ptr : msg.ptr+4])
	msg.ptr += 4
	return true
}

// 读取消息的payload
func (msg *LevinProtocolMessage) readPayload(conn *net.Conn) bool {
	// 读取payload
	// 先读取payload开头的
	buffer := make([]byte, 2048) // 在这里先设置缓冲区的大小为2048
	// 对端发送的数据存在两种情况，一种是发送的数据长度比较小，tcp的一个报文就可以传过来，这个时候message_length长度的缓冲区就可以直接拿下
	// 第二种情况是，传送的数据量较大，即message_length大于单个tcp报文的最大长度，就需要传多次
	for uint64(len(msg.payload_bytes)) < msg.length {
		n, err := (*conn).Read(buffer)
		if err != nil {
			log.Println("Error reading remaining levin message from connection:"+(*conn).RemoteAddr().String(), err)
		}
		msg.payload_bytes = append(msg.payload_bytes, buffer[:n]...)
	}

	// 读取完毕
	// 1. 先检查msg payload的前9个字节是否等于签名值
	msg.ptr = uint64(0)
	if !bytes.Equal(msg.payload_bytes[msg.ptr:msg.ptr+4], portableStorageSignature1) {
		log.Println("Error portable storage signature1!")
		return false
	}
	msg.ptr += 4
	if !bytes.Equal(msg.payload_bytes[msg.ptr:msg.ptr+4], portableStorageSignature2) {
		log.Println("Error portable storage signature2!")
		return false
	}
	msg.ptr += 4
	if msg.payload_bytes[msg.ptr] != portableStorageFormatVer {
		log.Println("Error portable storage format ver!")
		return false
	}
	msg.ptr += 1
	// 2. 检查payload真正的数据部分，递归的反序列化
	msg.payload = msg.readSection()
	return true
}

// 获取字符串的长度，这个字符串可能是键名，也可以是数据
func (msg *LevinProtocolMessage) getKeyNum() uint64 {
	key_num_byte := msg.payload_bytes[msg.ptr]
	key_num_mask := key_num_byte & portableRawSizeMarkMask // 取key_num_byte的最后两位进行检查

	var key_num uint64
	if key_num_mask == portableRawSizeMarkByte {
		key_num = uint64(key_num_byte) >> 2 // key_num_byte的高6位即为数据的长度
		msg.ptr++
	} else if key_num_mask == portableRawSizeMarkWord {
		key_num = uint64(binary.BigEndian.Uint16(msg.payload_bytes[msg.ptr:msg.ptr+2])) >> 2
		msg.ptr += 2
	} else if key_num_mask == portableRawSizeMarkDword {
		key_num = uint64(binary.BigEndian.Uint32(msg.payload_bytes[msg.ptr:msg.ptr+4])) >> 2
		msg.ptr += 3
	} else if key_num_mask == portableRawSizeMarkInt64 {
		key_num = binary.BigEndian.Uint64(msg.payload_bytes[msg.ptr:msg.ptr+8]) >> 2
		msg.ptr += 4
	} else {
		log.Println("Invalid key num!")
	}
	return key_num
}

// 读取payload数据签名之后的数据部分，一个section包含多个entry，一个entry中可能也包含多个section
func (msg *LevinProtocolMessage) readSection() map[string]interface{} {
	section := make(map[string]interface{})
	// 2.1 读取payload数据部分的键值对字段数
	key_num := msg.getKeyNum()
	// 根据获取的字段数，一个一个的反序列化
	for key_num > 0 {
		// read name length
		key_name_length := uint8(msg.payload_bytes[msg.ptr])
		msg.ptr++
		// read name
		key_name := string(msg.payload_bytes[msg.ptr : msg.ptr+uint64(key_name_length)])
		msg.ptr += uint64(key_name_length)

		section[key_name] = msg.readEntry()
		key_num--
	}
	return section
}

// 读取payload中的entry，entry可能是简单数据数组、entry数组或者简单数据
func (msg *LevinProtocolMessage) readEntry() interface{} {
	entry_type := msg.payload_bytes[msg.ptr]
	msg.ptr++
	if (entry_type & serializeFlagArray) != 0 {
		// 数据为简单数组数据，解析数组
		return msg.readArrayEntry(entry_type)
	} else if entry_type == serializeTypeArray {
		// 数据为Entry数组数据，需要为每一个Entry都再调用readEntry方法
		return msg.readEntryArrayEntry()
	} else {
		// 数据为单一数据，解析单一数据
		return msg.read(entry_type, 0)
	}
}

// 读取简单数据数组
func (msg *LevinProtocolMessage) readArrayEntry(entry_type byte) interface{} {
	entry_type &= ^serializeFlagArray // entry_type和serializeFlagArray按位取反的结果相与
	key_num := msg.getKeyNum()
	array := make([]interface{}, key_num)
	for key_num > 0 {
		array[uint64(len(array))-key_num] = msg.read(entry_type, 0)
		key_num--
	}
	return array
}

// 读取entry数组
func (msg *LevinProtocolMessage) readEntryArrayEntry() interface{} {
	entry_type := msg.payload_bytes[msg.ptr]
	msg.ptr++
	if (entry_type & serializeFlagArray) != 0 {
		log.Println("wrong type sequences")
	}
	return msg.readArrayEntry(entry_type)
}

// 读取简单数据
func (msg *LevinProtocolMessage) read(entry_type byte, count byte) interface{} {
	// 可以在调用处使用断言来区分实际返回的类型
	if entry_type == 0 && count > 0 {
		data := msg.payload_bytes[msg.ptr : msg.ptr+uint64(count)]
		msg.ptr += uint64(count)
		return data
	}

	if entry_type == serializeTypeUint64 {
		data := binary.LittleEndian.Uint64(msg.payload_bytes[msg.ptr : msg.ptr+8])
		msg.ptr += 8
		return data
	}
	if entry_type == serializeTypeInt64 {
		data := int64(binary.LittleEndian.Uint64(msg.payload_bytes[msg.ptr : msg.ptr+8]))
		msg.ptr += 8
		return data
	}
	if entry_type == serializeTypeUint32 {
		data := binary.LittleEndian.Uint32(msg.payload_bytes[msg.ptr : msg.ptr+4])
		msg.ptr += 4
		return data
	}
	if entry_type == serializeTypeInt32 {
		data := int32(binary.LittleEndian.Uint32(msg.payload_bytes[msg.ptr : msg.ptr+4]))
		msg.ptr += 4
		return data
	}
	if entry_type == serializeTypeUint16 {
		data := binary.LittleEndian.Uint16(msg.payload_bytes[msg.ptr : msg.ptr+2])
		msg.ptr += 2
		return data
	}
	if entry_type == serializeTypeInt16 {
		data := int16(binary.LittleEndian.Uint16(msg.payload_bytes[msg.ptr : msg.ptr+2]))
		msg.ptr += 2
		return data
	}
	if entry_type == serializeTypeUint8 {
		data := uint8(msg.payload_bytes[msg.ptr])
		msg.ptr++
		return data
	}
	if entry_type == serializeTypeInt8 {
		data := int8(msg.payload_bytes[msg.ptr])
		msg.ptr++
		return data
	}
	if entry_type == serializeTypeObject {
		return msg.readSection()
	}
	if entry_type == serializeTypeString {
		key_num := msg.getKeyNum()
		return msg.read(0, byte(key_num))
	}
	return nil
}

/*
==============

	序列化

==============
*/

// 写header（实际在对消息进行序列化时，应该先序列化payload，确定好payload的长度之后再构建头部）
func (msg *LevinProtocolMessage) writeHeader(message_type uint32, payload_length uint64, request bool) {
	msg.header_bytes = make([]byte, 0)
	msg.signature = binary.BigEndian.Uint64(levinSignature)
	msg.command = message_type
	if request {
		msg.expect_response = true
		msg.return_code = levinRequestReturnCode
		msg.flags = levinPacketRequest
	} else {
		msg.expect_response = false
		msg.return_code = levinResponseReturnCode
		msg.flags = levinPacketResponse
	}
	msg.version = levinProtocolVer1
	msg.length = payload_length

	// 对消息头部的序列化
	// 1. 添加Signature
	msg.header_bytes = binary.BigEndian.AppendUint64(msg.header_bytes, msg.signature)
	// 2. 添加Length
	msg.header_bytes = binary.LittleEndian.AppendUint64(msg.header_bytes, msg.length)
	// 3. 添加E.Response
	if msg.expect_response {
		msg.header_bytes = append(msg.header_bytes, uint8(1))
	} else {
		msg.header_bytes = append(msg.header_bytes, uint8(0))
	}
	// 4. 添加Command
	msg.header_bytes = binary.LittleEndian.AppendUint32(msg.header_bytes, msg.command)
	// 5. 添加Return Code
	msg.header_bytes = binary.LittleEndian.AppendUint32(msg.header_bytes, uint32(msg.return_code))
	// 6. 添加flags
	msg.header_bytes = binary.LittleEndian.AppendUint32(msg.header_bytes, msg.flags)
	// 7. 添加version
	msg.header_bytes = binary.LittleEndian.AppendUint32(msg.header_bytes, msg.version)
}

// 写payload，接收传入的map[string]interface{}，对键值对进行反序列化
func (msg *LevinProtocolMessage) writePayload(payload map[string]interface{}) {
	msg.payload_bytes = make([]byte, 0)
	// 先写payload的头部
	msg.payload_bytes = append(msg.payload_bytes, portableStorageSignature1...)
	msg.payload_bytes = append(msg.payload_bytes, portableStorageSignature2...)
	msg.payload_bytes = append(msg.payload_bytes, portableStorageFormatVer)
	// 开始写余下的内容
	msg.writeSection(payload)
}

func (msg *LevinProtocolMessage) setKeyNum(keyNum uint64) {
	if keyNum <= 63 {
		out := (byte(keyNum) << 2) | portableRawSizeMarkByte
		msg.payload_bytes = append(msg.payload_bytes, byte(out))
	} else if keyNum <= 16383 {
		out := (uint16(keyNum) << 2) | uint16(portableRawSizeMarkWord)
		msg.payload_bytes = binary.LittleEndian.AppendUint16(msg.payload_bytes, out)
	} else if keyNum <= 1073741823 {
		out := (uint32(keyNum) << 2) | (uint32(portableRawSizeMarkDword))
		msg.payload_bytes = binary.LittleEndian.AppendUint32(msg.payload_bytes, out)
	} else if keyNum <= 4611686018427387903 {
		out := (keyNum << 2) | uint64(portableRawSizeMarkInt64)
		msg.payload_bytes = binary.LittleEndian.AppendUint64(msg.payload_bytes, out)
	} else {
		log.Fatalln("failed to pack varint - too big amount")
	}
}

func (msg *LevinProtocolMessage) writeSection(data map[string]interface{}) {
	// 将Payload看作一整个Seciton
	msg.setKeyNum(uint64(len(data))) // 先写section的长度字段
	for key, value := range data {
		// 写键值对的键字符串的长度
		msg.payload_bytes = append(msg.payload_bytes, byte(len(key)))
		// 写键值对的键字符串
		msg.payload_bytes = append(msg.payload_bytes, []byte(key)...)
		msg.write(value)
	}
}

func (msg *LevinProtocolMessage) writeSectionEntry(data map[string]interface{}) {
	msg.payload_bytes = append(msg.payload_bytes, serializeTypeObject)
	msg.writeSection(data)
}

func (msg *LevinProtocolMessage) write(data interface{}) {
	switch data := data.(type) {
	case uint64:
		msg.payload_bytes = append(msg.payload_bytes, serializeTypeUint64)
		msg.payload_bytes = binary.LittleEndian.AppendUint64(msg.payload_bytes, data)
	case int64:
		msg.payload_bytes = append(msg.payload_bytes, serializeTypeInt64)
		msg.payload_bytes = binary.LittleEndian.AppendUint64(msg.payload_bytes, uint64(data))
	case uint32:
		msg.payload_bytes = append(msg.payload_bytes, serializeTypeUint32)
		msg.payload_bytes = binary.LittleEndian.AppendUint32(msg.payload_bytes, data)
	case int32:
		msg.payload_bytes = append(msg.payload_bytes, serializeTypeInt32)
		msg.payload_bytes = binary.LittleEndian.AppendUint32(msg.payload_bytes, uint32(data))
	case uint16:
		msg.payload_bytes = append(msg.payload_bytes, serializeTypeUint16)
		msg.payload_bytes = binary.LittleEndian.AppendUint16(msg.payload_bytes, data)
	case int16:
		msg.payload_bytes = append(msg.payload_bytes, serializeTypeInt16)
		msg.payload_bytes = binary.LittleEndian.AppendUint16(msg.payload_bytes, uint16(data))
	case uint8:
		msg.payload_bytes = append(msg.payload_bytes, serializeTypeUint8, data)
	case int8:
		msg.payload_bytes = append(msg.payload_bytes, serializeTypeInt8, uint8(data))
	case string:
		msg.payload_bytes = append(msg.payload_bytes, serializeTypeString)
		msg.setKeyNum(uint64(len(data)))
		msg.payload_bytes = append(msg.payload_bytes, []byte(data)...)
	case map[string]interface{}:
		msg.writeSectionEntry(data)
	case []interface{}:
		msg.payload_bytes = append(msg.payload_bytes, byte(0x8c))
		msg.setKeyNum(uint64(len(data)))
		for i := 0; i < len(data); i++ {
			msg.writeSection(data[i].(map[string]interface{}))
		}
	default:
		log.Fatalln("Unable to cast input to serialized data")
	}
}

func (msg *LevinProtocolMessage) writeHandshakeRequestPayload(my_port uint32, network_id []byte, peer_id uint64) map[string]interface{} {
	var genesis_hash []byte
	if bytes.Equal(network_id, network_id_mainnet) {
		genesis_hash = genesis_hash_mainnet
	} else if bytes.Equal(network_id, network_id_testnet) {
		genesis_hash = genesis_hash_testnet
	}
	data := make(map[string]interface{})
	// node_data
	node_data := make(map[string]interface{})
	node_data["my_port"] = my_port
	node_data["network_id"] = string(network_id)
	node_data["peer_id"] = peer_id
	node_data["support_flags"] = p2pSupportFlags
	data["node_data"] = node_data
	// payload_data
	payload_data := make(map[string]interface{})
	payload_data["cumulative_difficulty"] = uint64(0)
	payload_data["cumulative_difficulty_top64"] = uint64(0)
	payload_data["current_height"] = uint64(0)
	payload_data["top_id"] = string(genesis_hash)
	payload_data["top_version"] = byte(1)
	data["payload_data"] = payload_data

	return data
}

func (msg *LevinProtocolMessage) writeHandshakeResponsePayload(my_port uint32, network_id []byte, peer_id uint64, peerlist []interface{}) map[string]interface{} {
	var genesis_hash []byte
	if bytes.Equal(network_id, network_id_mainnet) {
		genesis_hash = genesis_hash_mainnet
	} else if bytes.Equal(network_id, network_id_testnet) {
		genesis_hash = genesis_hash_testnet
	}
	data := make(map[string]interface{})
	// local_peerlist_new
	data["local_peerlist_new"] = peerlist
	// node_data
	node_data := make(map[string]interface{})
	node_data["my_port"] = my_port
	node_data["network_id"] = string(network_id)
	node_data["peer_id"] = peer_id
	node_data["support_flags"] = p2pSupportFlags
	data["node_data"] = node_data
	// payload_data
	payload_data := make(map[string]interface{})
	payload_data["cumulative_difficulty"] = uint64(0)
	payload_data["cumulative_difficulty_top64"] = uint64(0)
	payload_data["current_height"] = uint64(0)
	payload_data["top_id"] = string(genesis_hash)
	payload_data["top_version"] = byte(1)
	data["payload_data"] = payload_data

	return data
}
