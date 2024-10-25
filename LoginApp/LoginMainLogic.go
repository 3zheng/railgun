package main

import (
	"fmt"

	"github.com/3zheng/railgun/PoolAndAgent"
	bs_client "github.com/3zheng/railgun/protodefine/client"
	bs_types "github.com/3zheng/railgun/protodefine/mytype"
	bs_router "github.com/3zheng/railgun/protodefine/router"
	proto "google.golang.org/protobuf/proto"
)

type LoginMainLogic struct {
	mLogicPool    *PoolAndAgent.SingleMsgPool //自身绑定的SingleMsgPool
	mDBPool       *PoolAndAgent.SingleMsgPool //数据库的pool
	mRouterAgents *PoolAndAgent.RouterAgent   //暂时一个router agent，以后可能会有多个
	mMyAppid      uint32
}

// 实现PoolAndAgent.ILogicProcess的三个接口函数
func (this *LoginMainLogic) Init(myPool *PoolAndAgent.SingleMsgPool) bool {
	this.mLogicPool = myPool
	return true
}

func (this *LoginMainLogic) ProcessReq(req proto.Message, pDatabase *PoolAndAgent.CADODatabase) {
	msg := Login_CreateCommonMsgByRouterTransferData(req)
	switch data := msg.(type) {
	case *PrivateInitMsg:
		this.Private_OnInit(data)
	case *bs_client.LoginReq:
		this.Client_OnLoginReq(data)
	case *bs_client.LoginRsp:
		this.Client_OnLoginRsp(data)
	default:
		return
	}
}

func (this *LoginMainLogic) OnPulse(ms uint64) {
	//定时调用程序
}

func (this *LoginMainLogic) Private_OnInit(req *PrivateInitMsg) {
	this.mMyAppid = req.myAppId
	this.mDBPool = req.pDBPool
}

func (this *LoginMainLogic) Client_OnLoginReq(req *bs_client.LoginReq) {
	fmt.Println("收到了登录验证请求")
	//登录验证报文直接丢给DBPool
	this.PushToDBPool(req)
}

func (this *LoginMainLogic) Client_OnLoginRsp(req *bs_client.LoginRsp) {
	fmt.Println("收到了登录验证回复")
	//直接把回复发往相应gate
	this.SendToOtherApp(req, req.Base)
}

// 向客户端发送
func (this *LoginMainLogic) SendToUserClient(req proto.Message, pBase *bs_types.BaseInfo, userId uint64, gateConnId uint64) {
	//向客户端发送消息，要先转为bs_router.RouterTransferData,让router中转到gate
	msg := Login_CreateRouterTransferDataByCommonMsg(req, pBase)
	switch msg := msg.(type) {
	case *bs_router.RouterTransferData:
		msg.DestAppid = pBase.AttAppid
		msg.DestApptype = uint32(bs_types.EnumAppType_Gate)
		msg.DataDirection = bs_router.RouterTransferData_App2Client //发往用户客户端
		msg.ClientRemoteAddress = pBase.RemoteAdd
		msg.AttGateid = pBase.AttAppid
		msg.AttUserid = userId
		msg.AttGateconnid = gateConnId
	}
	//这里只有SendMsgToServerAppByRouter，因为并没有绑定netagent
	this.mLogicPool.SendMsgToServerAppByRouter(msg)
}

// 向某个APP发送
func (this *LoginMainLogic) SendToOtherApp(req proto.Message, pBase *bs_types.BaseInfo) {
	//向客户端发送消息，要先转为bs_router.RouterTransferData,让router中转到gate
	msg := Login_CreateRouterTransferDataByCommonMsg(req, pBase)
	switch msg := msg.(type) {
	case *bs_router.RouterTransferData:
		msg.DestAppid = pBase.AttAppid
		msg.DestApptype = pBase.AttApptype
		msg.DataDirection = bs_router.RouterTransferData_App2App //发往其他app
		fmt.Println("RouterTransferData的DestAppid=", msg.DestAppid, ",DestApptype=", msg.DestApptype, "msg.DataDirection=", msg.DataDirection)
	}
	//这里只有SendMsgToServerAppByRouter，因为并没有绑定netagent
	this.mLogicPool.SendMsgToServerAppByRouter(msg)
}

// 向DBPOOL发送（伪），这个发送实际上不走TCP/IP，是程序内部间的“发送”
func (this *LoginMainLogic) PushToDBPool(req proto.Message) {
	if this.mDBPool != nil {
		this.mDBPool.PushMsg(req, 0) //往mDBPool的队列尾推入消息
	}
}
