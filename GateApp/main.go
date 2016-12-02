// GateApp project main.go
package main

import (
	"fmt"

	"github.com/3zheng/railgun/PoolAndAgent"
	bs_types "github.com/3zheng/railgun/protodefine/mytype"
)

func CreateGateLogicInstance() *GateLogic {
	return new(GateLogic)
}

func main() {
	//先创建需要的变量
	quit := make(chan int)
	var myAppId uint32 = 101
	pNetAgent := PoolAndAgent.CreateNetAgent("0.0.0.0:4101") //监听4101端口
	pLogicPool := PoolAndAgent.CreateMsgPool(quit, uint32(bs_types.EnumAppType_Gate), myAppId)
	pRouterAgent := PoolAndAgent.CreateRouterAgent("127.0.0.1:2001") //连接127.0.0.1:2001地址
	pGateLogic := CreateGateLogicInstance()
	//将他们都与Pool绑定起来
	pLogicPool.AddLogicProcess(pGateLogic)
	pLogicPool.BindNetAgent(pNetAgent)
	pLogicPool.BindRouterAgent(pRouterAgent)
	//运行
	ok := pLogicPool.InitAndRun(nil)
	if ok {
		fmt.Println("初始化完毕")
		pInitMsg := &PrivateInitMsg{
			pNetAgent:    pNetAgent,
			pRouterAgent: pRouterAgent,
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
