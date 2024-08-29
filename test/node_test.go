package test

import (
	"gomonero/node"
	"testing"
)

func Test_Node(t *testing.T) {
	node := node.CreateNode(38080)
	node.Start()
}
