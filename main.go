package main

import (
	"gomonero/node"
	"gomonero/web"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func main() {
	// 启动节点
	node := node.CreateNode("testnet", 28083)
	node.Start()
	defer node.Stop()

	// 启动Gin框架，等待http请求
	r := gin.Default()

	/*
		========================
		blockchain_explorer路由组
		========================
	*/
	blockchain_explorer_router := r.Group("/blockchain_explorer")
	blockchain_explorer_router.GET("/block", func(c *gin.Context) {

	})

	/*
		====================
		eclipse_attack路由组
		====================
	*/
	eclipse_attack_router := r.Group("/eclipse_attack")
	// graylist attack
	eclipse_attack_router.GET("/graylist_attack", func(c *gin.Context) {
		target_ip := c.Query("target_ip")
		target_port, err := strconv.Atoi(c.Query("target_port"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"Error":  err.Error(),
				"status": "Failed",
			})
		}
		err = web.GraylistAttack(node, target_ip, uint16(target_port))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"Error":  err.Error(),
				"status": "Failed",
			})
		} else {
			c.JSON(http.StatusOK, gin.H{
				"message": "Graylist attack started!",
				"status":  "OK",
			})
		}
	})
	// whitelist attack
	eclipse_attack_router.GET("/whitelist_attack", func(c *gin.Context) {
		target_ip := c.Query("target_ip")
		target_port, err := strconv.Atoi(c.Query("target_port"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"Error":  err.Error(),
				"status": "Failed",
			})
		}
		err = web.WhitelistAttack(node, target_ip, uint16(target_port))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"Error":  err.Error(),
				"status": "Failed",
			})
		} else {
			c.JSON(http.StatusOK, gin.H{
				"message": "Whitelist attack started!",
				"status":  "OK",
			})
		}
	})

	/*
		===================
		monero_wallet路由组
		===================
	*/
	monero_wallet_router := r.Group("/monero_wallet")
	monero_wallet_router.GET("/balance", func(c *gin.Context) {

	})

	// 路由错误时提示404 Not Found
	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, "404 Not Found!")
	})
	r.Run()
}
