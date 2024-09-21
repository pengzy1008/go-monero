package levin

import (
	"encoding/binary"
	"log"
)

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
