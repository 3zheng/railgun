package main

import (
	"github.com/3zheng/railgun/PoolAndAgent"
	proto "github.com/golang/protobuf/proto"
)

/*这个源文件的作用：
//自己构建用来login内部使用的私有报文
//为了能在pool传输必须继承proto.Message,所以要重写以下3个函数
//Reset()
//String() string
//ProtoMessage()
//但是由于不是由protoc.exe来产生的正常报文，是无法被proto解析的
*/

//初始化报文
type PrivateInitMsg struct {
	pNetAgent    *PoolAndAgent.NetAgent
	pRouterAgent *PoolAndAgent.RouterAgent
	pMainPool    *PoolAndAgent.SingleMsgPool
	pDBPool      *PoolAndAgent.SingleMsgPool
	myAppId      uint32
}

//均为空
func (*PrivateInitMsg) Reset() {

}

func (*PrivateInitMsg) String() string {
	return "PrivateInitMsg"
}

func (*PrivateInitMsg) ProtoMessage() {

}

//当逻辑层需要区别这个报文是自己延时推送给自己的报文还是其他人推送过来的报文的时候，
//可以用这个私有类型,pDelay用来装延时发送的实际报文指针，DelayTime是延时时间
//当收到这个报文时，说明是自己向自己延时推送的
type PrivateDelayMsg struct {
	pDelay    proto.Message //延时发送的报文
	DelayTime uint64        //延时发送的时间
}

//均为空
func (*PrivateDelayMsg) Reset() {

}

func (*PrivateDelayMsg) String() string {
	return "PrivateDelayMsg"
}

func (*PrivateDelayMsg) ProtoMessage() {

}
