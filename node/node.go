package node

// 虚拟的门罗币节点
type Node struct {
	my_port uint16
	peer_id string
}

func (node Node) Start() {
	// IO多路复用启动Server

}
