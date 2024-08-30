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

// payload header
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
	signature   uint64
	length      uint64
	return_data bool
	command     uint32
	return_code int32
	flags       uint32
	version     uint32

	// payload的反序列化后的字段
	payload map[string]interface{}
}

/*
===============

	反序列化

===============
*/
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
	msg.return_data = (msg.header_bytes[msg.ptr] != 0)
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

func (msg *LevinProtocolMessage) readEntryArrayEntry() interface{} {
	entry_type := msg.payload_bytes[msg.ptr]
	msg.ptr++
	if (entry_type & serializeFlagArray) != 0 {
		log.Println("wrong type sequences")
	}
	return msg.readArrayEntry(entry_type)
}

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
func (msg *LevinProtocolMessage) WriteHeader() {

}

func (msg *LevinProtocolMessage) WritePayload() {

}
