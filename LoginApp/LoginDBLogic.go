package main

import (
	"fmt"

	"github.com/3zheng/railgun/PoolAndAgent"
	protodf "github.com/3zheng/railgun/protodf"
	protodf "github.com/3zheng/railgun/protodf/client"
	proto "google.golang.org/protobuf/proto"
)

type LoginDBLogic struct {
	mLogicPool    *PoolAndAgent.SingleMsgPool //主逻辑线程的Pool
	mDBPool       *PoolAndAgent.SingleMsgPool //自身绑定的SingleMsgPool
	mRouterAgents *PoolAndAgent.RouterAgent   //暂时一个router agent，以后可能会有多个
	mMyAppid      uint32
}

// 实现PoolAndAgent.ILogicProcess的三个接口函数
func (this *LoginDBLogic) Init(myPool *PoolAndAgent.SingleMsgPool) bool {
	this.mDBPool = myPool
	return true
}

func (this *LoginDBLogic) ProcessReq(req proto.Message, pDatabase *PoolAndAgent.CADODatabase) {
	//因为DBPool的报文来源都是主逻辑Pool,并没有直接绑定routerAgent，所以收到的全部都是普通报文。不需要调用Login_CreateCommonMsgByRouterTransferData
	switch data := req.(type) {
	case *PrivateInitMsg:
		this.Private_OnInit(data)
	case *protodf.LoginReq:
		this.Client_OnDBLoginReq(data, pDatabase)
	default:
		return
	}
}

func (this *LoginDBLogic) OnPulse(ms uint64) {
	//定时调用程序
}

func (this *LoginDBLogic) Private_OnInit(req *PrivateInitMsg) {
	this.mMyAppid = req.myAppId
	this.mLogicPool = req.pMainPool
}

func (this *LoginDBLogic) Client_OnDBLoginReq(req *protodf.LoginReq, pDatabase *PoolAndAgent.CADODatabase) {
	sqlExpress := fmt.Sprintf("select * from user_base where login_account = '%s' and passwd ='%s'", req.LoginAccount, req.LoginPassword)
	pDatabase.ReadFromDB(sqlExpress)

	rsp := new(protodf.LoginRsp)
	protodf.SetBaseKindAndSubId(rsp)
	protodf.CopyBaseExceptKindAndSubId(rsp.Base, req.Base)
	rsp.UserSesionInfo.Client_IP = req.Base.RemoteAdd
	protodf.OutputMyLog("login RemoteAdd=", req.Base.RemoteAdd)
	if pDatabase.ReadInfo.RowNum != 0 {
		var userId uint64
		var nickName string
		pDatabase.GetValueByRowIdAndColName(0, "userid", &userId)
		rsp.UserBaseInfo.UserId = userId
		rsp.UserId = userId
		pDatabase.GetValueByRowIdAndColName(0, "nick_name", &nickName)
		rsp.UserBaseInfo.NickName = nickName
		rsp.LoginResult = protodf.LoginRsp_SUCCESS //登录成功
		fmt.Println("登录成功,userId=", userId)
	} else {
		fmt.Println("返回行数为0，密码或者账号错误")
		rsp.LoginResult = protodf.LoginRsp_FALSEPW
	}
	//把登录回复传给主逻辑线程处理
	this.PushToMainPool(rsp)
}

// 向DBPOOL发送（伪），这个发送实际上不走TCP/IP，是程序内部间的“发送”
func (this *LoginDBLogic) PushToMainPool(req proto.Message) {
	if this.mLogicPool != nil {
		this.mLogicPool.PushMsg(req, 0) //往mDBPool的队列尾推入消息
	}
}
