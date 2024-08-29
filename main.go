package main

import "gomonero/rpcproxy"

func main() {
	wallet_proxy := rpcproxy.NewWalletRPCProxy(
		"127.0.0.1",
		28088,
		"pengzy1008",
		"123456",
	)
	wallet_proxy.GetBalance(0, nil, false, false)
}
