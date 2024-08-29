package test

import (
	"gomonero/rpcproxy"
	"testing"
)

func Test_DaemonRPCProxy_GetBlockCount(t *testing.T) {
	daemon_proxy := rpcproxy.CreateDaemonRPCProxy(
		"43.138.89.105",
		28081,
		"pengzy1008",
		"123456",
	)
	block_count := daemon_proxy.GetBlockCount()
	expected := (block_count != -1)
	if !expected {
		t.Errorf("Test_DaemonRPCProxy_GetBlockCount failed!")
	}
}

func Test_DaemonRPCProxy_GetHeight(t *testing.T) {
	daemon_proxy := rpcproxy.CreateDaemonRPCProxy(
		"43.138.89.105",
		28081,
		"pengzy1008",
		"123456",
	)
	block_height := daemon_proxy.GetHeight()
	expected := (block_height != -1)
	if !expected {
		t.Errorf("Test_DaemonRPCProxy_GetBlockCount failed!")
	}
}
