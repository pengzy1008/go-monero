package web

import "gomonero/node"

// 灰名单攻击
func GraylistAttack(node *node.Node, target_ip string, target_port uint16) error {
	return node.EstablishOutgoingConnection(target_ip, target_port, false)
}

// 白名单攻击
func WhitelistAttack(node *node.Node, target_ip string, target_port uint16) error {
	return node.EstablishOutgoingConnection(target_ip, target_port, true)
}
