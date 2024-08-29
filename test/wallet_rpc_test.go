package test

import (
	"gomonero/rpcproxy"
	"testing"
)

func Test_WalletRPCProxy_GetBalance(t *testing.T) {
	wallet_proxy := rpcproxy.NewWalletRPCProxy(
		"127.0.0.1",
		28088,
		"pengzy1008",
		"123456",
	)
	balance, unlocked_balance := wallet_proxy.GetBalance(0, nil, false, false)
	expected := !(balance == -1 || unlocked_balance == -1)
	if !expected {
		t.Errorf("Test_WalletRPCProxy_GetBalance failed!")
	}
}

func Test_WalletRPCProxy_Transfer(t *testing.T) {
	wallet_proxy := rpcproxy.NewWalletRPCProxy(
		"127.0.0.1",
		28088,
		"pengzy1008",
		"123456",
	)
	destinations := []map[string]interface{}{
		{
			"amount":  uint64(100000000000),
			"address": "9yZhA4eVVjBd6ihbdTTifB2BxDn2UiLKuY79Y13VdxDt7kRzpNkV3HS3XvjcbFEsz2hqUF7dzUSthN6Ea2wF6mpPVbXzsiX",
		},
	}
	result := wallet_proxy.Transfer(destinations, 0, nil, false, false, true, true)
	expected := result != nil
	if !expected {
		t.Errorf("Test_WalletRPCProxy_Transfer failed!")
	}
}
