package rpcproxy

import (
	"log"
	"strconv"
)

type DaemonRPCProxy struct {
	rpc_ip        string
	rpc_port      int16
	rpc_user      string
	rpc_password  string
	json_rpc_url  string
	other_rpc_url string
}

// 创建DaemonRPC代理，所有的和节点有关的RPC调用都通过它来完成
func CreateDaemonRPCProxy(rpc_ip string, rpc_port int16, rpc_user string, rpc_password string) DaemonRPCProxy {
	proxy := DaemonRPCProxy{
		rpc_ip:       rpc_ip,
		rpc_port:     rpc_port,
		rpc_user:     rpc_user,
		rpc_password: rpc_password,
	}
	proxy.json_rpc_url = "http://" + rpc_ip + ":" + strconv.Itoa(int(rpc_port)) + "/json_rpc"
	proxy.other_rpc_url = "http://" + rpc_ip + ":" + strconv.Itoa(int(rpc_port))
	return proxy
}

/*
	================
	JSON RPC Methods
	================
*/

// get_block_height
func (proxy *DaemonRPCProxy) GetBlockCount() int {
	rpc_method := "get_block_count"
	params := map[string]interface{}{}
	response := JsonMethodRequest(proxy.json_rpc_url, rpc_method, params, proxy.rpc_user, proxy.rpc_password)
	if result, ok := response["result"].(map[string]interface{}); ok {
		block_count := int(result["count"].(float64))
		return block_count
	}
	log.Println("Error occur when get block count!")
	return -1
}

/*
	=================
	Other RPC Methods
	=================
*/

// get_height
func (proxy *DaemonRPCProxy) GetHeight() int {
	rpc_method := "get_height"
	params := map[string]interface{}{}
	response := OtherMethodRequest(proxy.other_rpc_url, rpc_method, params, proxy.rpc_user, proxy.rpc_password)
	if result, ok := response["height"].(float64); ok {
		return int(result)
	}
	log.Println("Error occur when get block height!")
	return -1
}
