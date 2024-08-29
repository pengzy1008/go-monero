package main

import "gomonero/node"

func main() {
	node := node.CreateNode(38080)
	node.Start()
}
