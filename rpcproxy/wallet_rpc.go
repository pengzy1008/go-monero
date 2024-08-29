package rpcproxy

import (
	"fmt"
	"log"
	"strconv"
)

type WalletRPCProxy struct {
	rpc_ip        string
	rpc_port      int16
	rpc_user      string
	rpc_password  string
	json_rpc_url  string
	other_rpc_url string
}

// Transfer函数接收的destination数组的元素
type Destination struct {
	Amount  uint64
	Address string
}

func CreateWalletRPCProxy(rpc_ip string, rpc_port int16, rpc_user string, rpc_password string) WalletRPCProxy {
	proxy := WalletRPCProxy{
		rpc_ip:       rpc_ip,
		rpc_port:     rpc_port,
		rpc_user:     rpc_user,
		rpc_password: rpc_password,
	}
	proxy.json_rpc_url = "http://" + rpc_ip + ":" + strconv.Itoa(int(rpc_port)) + "/json_rpc"
	proxy.other_rpc_url = "http://" + rpc_ip + ":" + strconv.Itoa(int(rpc_port))
	return proxy
}

// get_balance: 除第一个参数外，其余参数均可以设置为nil或false
func (proxy WalletRPCProxy) GetBalance(
	account_index uint, address_indices []uint, all_acounts bool, strict bool) (balance int64, unlocked_balance int64) {

	rpc_method := "get_balance"
	params := map[string]interface{}{
		"account_index": account_index,
		"all_accounts":  all_acounts,
		"strict":        strict,
	}
	// if address_indices != nil {
	// 	params["address_indices"] = address_indices
	// }
	response := JsonMethodRequest(proxy.json_rpc_url, rpc_method, params, proxy.rpc_user, proxy.rpc_password)
	if result, ok := response["result"].(map[string]interface{}); ok {
		balance := int64(result["balance"].(float64))
		unlocked_balance := int64(result["unlocked_balance"].(float64))
		return balance, unlocked_balance
	}
	log.Println("Error occur when get balance!")
	log.Println("response:", response)
	return -1, -1
}

func (proxy WalletRPCProxy) Transfer(
	destinations []map[string]interface{}, account_index uint, subaddr_indices []uint, get_tx_key bool,
	do_not_relay bool, get_tx_hex bool, get_tx_metadata bool) map[string]interface{} {

	rpc_method := "transfer"
	params := map[string]interface{}{
		"destinations":    destinations,
		"account_index":   account_index,
		"subaddr_indices": subaddr_indices,
		"get_tx_key":      get_tx_key,
		"do_not_relay":    do_not_relay,
		"get_tx_hex":      get_tx_hex,
		"get_tx_metadata": get_tx_metadata,
	}
	response := JsonMethodRequest(proxy.json_rpc_url, rpc_method, params, proxy.rpc_user, proxy.rpc_password)
	if result, ok := response["result"].(map[string]interface{}); ok {
		fmt.Println(result)
		return result
	}
	log.Println("Error occur when transfer!")
	log.Println("response:", response)
	return nil
}
