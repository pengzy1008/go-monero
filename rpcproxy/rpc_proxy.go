package rpcproxy

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"

	digest_auth_client "github.com/xinsnake/go-http-digest-auth-client"
)

/*
================================
对：发送RPC请求，接收RPC响应的封装
================================
*/

// json rpc 请求
func JsonMethodRequest(
	json_rpc_url string, rpc_method string, params map[string]interface{},
	rpc_user string, rpc_password string) map[string]interface{} {

	request_data := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      "0",
		"method":  rpc_method,
		"params":  params,
	}
	json_data, err := json.Marshal(request_data)
	if err != nil {
		log.Println("Error: ", err)
		return nil
	}
	requset, err := http.NewRequest("POST", json_rpc_url, bytes.NewBuffer(json_data))
	if err != nil {
		log.Println("Error: ", err)
		return nil
	}
	requset.Header.Set("Content-Type", "application/json")

	client := digest_auth_client.NewTransport(rpc_user, rpc_password)
	resp, err := client.RoundTrip(requset)
	if err != nil {
		log.Println("Error: ", err)
		return nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("Error: ", err)
		return nil
	}
	response := map[string]interface{}{}
	json.Unmarshal(body, &response)
	return response
}

// other rpc 请求
func OtherMethodRequest(
	other_rpc_url string, rpc_method string, params map[string]interface{},
	rpc_user string, rpc_password string) map[string]interface{} {

	json_data, err := json.Marshal(params)
	if err != nil {
		log.Println("Error: ", err)
		return nil
	}
	requset, err := http.NewRequest("POST", other_rpc_url+"/"+rpc_method, bytes.NewBuffer(json_data))
	if err != nil {
		log.Println("Error: ", err)
		return nil
	}
	requset.Header.Set("Content-Type", "application/json")

	client := digest_auth_client.NewTransport(rpc_user, rpc_password)
	resp, err := client.RoundTrip(requset)
	if err != nil {
		log.Println("Error: ", err)
		return nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("Error: ", err)
		return nil
	}
	response := map[string]interface{}{}
	json.Unmarshal(body, &response)
	return response
}
