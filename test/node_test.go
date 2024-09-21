package test

import (
	"crypto/rand"
	"fmt"
	"gomonero/node"
	"math/big"
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

func Test_Peerlist(t *testing.T) {
	// local_peerlist_new := []interface{}{}
	// for index, value := range local_peerlist_new {
	// 	fmt.Println(index, value)
	// }
	// fmt.Println(len(local_peerlist_new))
	max := new(big.Int).Lsh(big.NewInt(1), 32)
	for i := 0; i < 10; i++ {
		random_num, _ := rand.Int(rand.Reader, max)
		fmt.Println(random_num.Uint64())
	}
}
