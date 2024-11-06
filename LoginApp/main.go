// LoginApp project main.go
package main

import (
	"fmt"

	"github.com/3zheng/railcommon"
	protodf "github.com/3zheng/railproto"
)

func CreateLoginLogicInstance() *LoginMainLogic {
	return new(LoginMainLogic)
}

func CreateLoginDBInstance() *LoginDBLogic {
	return new(LoginDBLogic)
}

func main() {
	//先创建需要的变量
	quit := make(chan int)
	var myAppId uint32 = 30
	pLogicPool := railcommon.CreateMsgPool(quit, uint32(protodf.EnumAppType_Login), myAppId)
	pRouterAgent := railcommon.CreateRouterAgent("127.0.0.1:2001") //连接127.0.0.1:2001地址
	pMainLoginLogic := CreateLoginLogicInstance()
	//将他们都与Pool绑定起来
	pLogicPool.AddLogicProcess(pMainLoginLogic)
	pLogicPool.BindRouterAgent(pRouterAgent)
	//创建数据库协程,只有主线程pLogicPool需要绑定RouterAgent，数据库协程是不需要的,所以退出quit也不需要创建，因为程序退出又逻辑主线程来控制
	//先创建需要的变量
	pDBPool := railcommon.CreateMsgPool(nil, uint32(protodf.EnumAppType_Login), myAppId)
	for i := 0; i < 10; i++ {
		//创建10个数据库协程，因为数据库IO速度比较慢会存在阻塞时间，所以要多开几个，
		//数据库逻辑协程间最好不要有数据通信，都应该通过主逻辑POOL与主逻辑进行通信
		//相对的主逻辑最好只用一个协程，这样写业务逻辑也好写，
		pDBLogic := CreateLoginDBInstance()
		pDBProcess := railcommon.CreateADODatabase("root:123456@tcp(localhost:3306)/gotest?charset=utf8")
		//将他们都与Pool绑定起来
		pDBPool.AddLogicProcess(pDBLogic)
		pDBPool.AddDataBaseProcess(pDBProcess)
	}

	//运行主逻辑线程pool
	ok := pLogicPool.InitAndRun(nil)
	if ok {
		fmt.Println("主逻辑POOL初始化完毕")
		//这里要在初始化完毕后把DBPool和MainPool的指针传过去，这样才能两个Pool之间才可以相互传递数据
		pInitMsg := &PrivateInitMsg{
			pNetAgent:    nil,
			pRouterAgent: pRouterAgent,
			myAppId:      myAppId,
			pMainPool:    pLogicPool,
			pDBPool:      pDBPool}
		//在初始化完毕后向逻辑层发送初始化报文，带着一些初始化信息，比如自己的APPID等
		pLogicPool.PushMsg(pInitMsg, 0)
	} else {
		return
	}
	//运行数据库协程pool
	pInitMsg := &PrivateInitMsg{
		pNetAgent:    nil,
		pRouterAgent: pRouterAgent,
		myAppId:      myAppId,
		pMainPool:    pLogicPool,
		pDBPool:      pDBPool}
	ok = pDBPool.InitAndRun(pInitMsg)
	if ok {
		fmt.Println("数据库逻辑POOL初始化完毕")
	} else {
		return
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
