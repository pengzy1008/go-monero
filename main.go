package main

import "gomonero/node"

func main() {
	node := node.CreateNode("testnet", 58080)
	node.Start()
}
