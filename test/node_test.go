package test

import (
	"gomonero/node"
	"testing"
)

func Test_NodeAcceptIncomingConnection(t *testing.T) {
	node := node.CreateNode("testnet", 38080)
	node.Start()
}

func Test_NodeSetOutgoingConnection(t *testing.T) {
	node := node.CreateNode("testnet", 48080)
	node.Start()
}
