// RouterApp project main.go
package main

import (
	"PoolAndAgent"
	"fmt"
	bs_types "protodefine/mytype"
)

func CreateRouterLogicInstance() *RouterLogic {
	return new(RouterLogic)
}

func main() {
	quit := make(chan int)
	var myAppId uint32 = 50
	pNetAgent := PoolAndAgent.CreateNetAgent("0.0.0.0:2001") //监听2001端口
	pLogicPool := PoolAndAgent.CreateMsgPool(quit, uint32(bs_types.EnumAppType_Router), myAppId)
	pRouterLogic := CreateRouterLogicInstance()
	//将他们都与Pool绑定起来
	pLogicPool.AddLogicProcess(pRouterLogic)
	pLogicPool.BindNetAgent(pNetAgent)
	//运行
	ok := pLogicPool.InitAndRun(nil)
	if ok {
		fmt.Println("初始化完毕,监听2001端口")
		pInitMsg := &PrivateInitMsg{
			pNetAgent:    pNetAgent,
			pRouterAgent: nil, //本身就是router不需要连接router的RouterAgent
			myAppId:      myAppId}
		//在初始化完毕后向逻辑层发送初始化报文，带着一些初始化信息，比如自己的APPID等
		pLogicPool.PushMsg(pInitMsg, 0)
	}
	//阻塞直到收到quit请求
	for {
		select {
		case v := <-quit:
			if v == 1 { //只有在收到1时才退出主线程
				return
			}
		}
	}
}
