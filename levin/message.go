package levin

import (
	"bytes"
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
var NetworkIdMainnet = []byte{0x12, 0x30, 0xf1, 0x71, 0x61, 0x04, 0x41, 0x61, 0x17, 0x31, 0x00, 0x82, 0x16, 0xa1, 0xa1, 0x10}
var NetworkIdTestnet = []byte{0x12, 0x30, 0xf1, 0x71, 0x61, 0x04, 0x41, 0x61, 0x17, 0x31, 0x00, 0x82, 0x16, 0xa1, 0xa1, 0x11}
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

type PeerlistEntry struct {
	IP     uint32
	Port   uint16
	PeerId uint64
}

const CommandHandshake = 1001
const CommandTimedSync = 1002
const CommandPingPong = 1003

/*
======================================

	Read Monero Protocol Message

======================================
*/

func (msg *LevinProtocolMessage) ReadBuffer(conn *net.Conn) bool {
	return msg.readHeader(conn) && msg.readPayload(conn)
}

func (msg *LevinProtocolMessage) GetCommand() uint32 {
	return msg.command
}

func (msg *LevinProtocolMessage) GetExpectResponse() bool {
	return msg.expect_response
}

func (msg *LevinProtocolMessage) HeaderBytes() []byte {
	return msg.header_bytes
}

func (msg *LevinProtocolMessage) PayloadBytes() []byte {
	return msg.payload_bytes
}

/*
======================================

	Create Monero Protocol Message

======================================
*/

func (msg *LevinProtocolMessage) CreateHandshakeRequest(my_port uint32, network_id []byte, peer_id uint64) {
	payload_map := msg.writeHandshakeRequestPayload(my_port, network_id, peer_id)
	msg.writePayload(payload_map)
	msg.writeHeader(CommandHandshake, uint64(len(msg.payload_bytes)), true)
}

func (msg *LevinProtocolMessage) CreateHandshakeResponse(my_port uint32, network_id []byte, peer_id uint64, peerlist []PeerlistEntry) {
	payload_map := msg.writeHandshakeResponsePayload(my_port, network_id, peer_id, peerlist)
	msg.writePayload(payload_map)
	msg.writeHeader(CommandHandshake, uint64(len(msg.payload_bytes)), false)
}

func (msg *LevinProtocolMessage) CreateTimedSyncRequest(network_id []byte) {
	payload_map := msg.writeTimedSyncRequestPayload(network_id)
	msg.writePayload(payload_map)
	msg.writeHeader(CommandTimedSync, uint64(len(msg.payload_bytes)), true)
}

func (msg *LevinProtocolMessage) CreateTimedSyncResponse(my_port uint32, network_id []byte, peer_id uint64, peerlist []PeerlistEntry) {
	payload_map := msg.writeTimedSyncResponsePayload(my_port, network_id, peer_id, peerlist)
	msg.writePayload(payload_map)
	msg.writeHeader(CommandTimedSync, uint64(len(msg.payload_bytes)), false)
}

func (msg *LevinProtocolMessage) CreatePingRequest() {
	msg.writeHeader(CommandPingPong, 0, true)
	// Ping Request msg has no payload
}

func (msg *LevinProtocolMessage) CreatePongResponse(peer_id uint64) {
	payload_map := msg.writePongResponsePayload(peer_id)
	msg.writePayload(payload_map)
	msg.writeHeader(CommandPingPong, uint64(len(msg.payload_bytes)), false)
}

/*
======================================

	Write Monero message Payload

======================================
*/

func (msg *LevinProtocolMessage) writeHandshakeRequestPayload(my_port uint32, network_id []byte, peer_id uint64) map[string]interface{} {
	var genesis_hash []byte
	if bytes.Equal(network_id, NetworkIdMainnet) {
		genesis_hash = genesis_hash_mainnet
	} else if bytes.Equal(network_id, NetworkIdTestnet) {
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

func (msg *LevinProtocolMessage) writeHandshakeResponsePayload(my_port uint32, network_id []byte, peer_id uint64, peerlist []PeerlistEntry) map[string]interface{} {
	var genesis_hash []byte
	if bytes.Equal(network_id, NetworkIdMainnet) {
		genesis_hash = genesis_hash_mainnet
	} else if bytes.Equal(network_id, NetworkIdTestnet) {
		genesis_hash = genesis_hash_testnet
	}
	data := make(map[string]interface{})
	// local_peerlist_new
	local_peerlist_new := []interface{}{}
	for _, peer_entry := range peerlist {
		peer := make(map[string]interface{})
		adr := make(map[string]interface{})
		addr := make(map[string]interface{})
		addr["m_ip"] = peer_entry.IP
		addr["m_port"] = peer_entry.Port
		peer_type := uint8(1)
		adr["addr"] = addr
		adr["type"] = peer_type
		peer["adr"] = adr
		peer["id"] = peer_entry.PeerId
		local_peerlist_new = append(local_peerlist_new, peer)
	}
	data["local_peerlist_new"] = local_peerlist_new
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

// Timed Sync 请求
func (msg *LevinProtocolMessage) writeTimedSyncRequestPayload(network_id []byte) map[string]interface{} {
	var genesis_hash []byte
	if bytes.Equal(network_id, NetworkIdMainnet) {
		genesis_hash = genesis_hash_mainnet
	} else if bytes.Equal(network_id, NetworkIdTestnet) {
		genesis_hash = genesis_hash_testnet
	}
	data := make(map[string]interface{})
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

// Timed Sync 响应
func (msg *LevinProtocolMessage) writeTimedSyncResponsePayload(my_port uint32, network_id []byte, peer_id uint64, peerlist []PeerlistEntry) map[string]interface{} {
	var genesis_hash []byte
	if bytes.Equal(network_id, NetworkIdMainnet) {
		genesis_hash = genesis_hash_mainnet
	} else if bytes.Equal(network_id, NetworkIdTestnet) {
		genesis_hash = genesis_hash_testnet
	}
	data := make(map[string]interface{})
	// local_peerlist_new
	// local_peerlist_new
	local_peerlist_new := []interface{}{}
	for _, peer_entry := range peerlist {
		peer := make(map[string]interface{})
		adr := make(map[string]interface{})
		addr := make(map[string]interface{})
		addr["m_ip"] = peer_entry.IP
		addr["m_port"] = peer_entry.Port
		peer_type := uint8(1)
		adr["addr"] = addr
		adr["type"] = peer_type
		peer["adr"] = adr
		peer["id"] = peer_entry.PeerId
		local_peerlist_new = append(local_peerlist_new, peer)
	}
	data["local_peerlist_new"] = local_peerlist_new
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

// Ping消息没有payload

// Pong消息有payload
func (msg *LevinProtocolMessage) writePongResponsePayload(peer_id uint64) map[string]interface{} {
	data := make(map[string]interface{})
	data["peer_id"] = peer_id
	data["status"] = string([]byte{0x4f, 0x4b})
	return data
}
