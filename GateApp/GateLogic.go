package main

import (
	"fmt"
	"time"

	"github.com/3zheng/railgun/PoolAndAgent"
	bs_proto "github.com/3zheng/railgun/protodefine"
	bs_client "github.com/3zheng/railgun/protodefine/client"
	bs_gate "github.com/3zheng/railgun/protodefine/gate"
	bs_types "github.com/3zheng/railgun/protodefine/mytype"
	bs_router "github.com/3zheng/railgun/protodefine/router"
	bs_tcp "github.com/3zheng/railgun/protodefine/tcpnet"
	proto "github.com/golang/protobuf/proto"
)

type GateConnection struct {
	connId           uint64
	isConnectted     bool
	clientAdress     string //IP地址含端口，格式192.168.1.1:10010
	userId           uint64 //对应的用户ID
	isAuthenticated  bool   //在登录成功后isAuthenticated为true
	connectedTime    int64  //time.Now().Unix()，建立起连接的时间
	lastResponseTime int64  //time.Now().Unix()，最近一次的响应时间，用于判断是否因为超时需要断开这个链接
	msgFromClientNum uint64 //来自这个连接的报文数，有恶意发包行为可以踢掉之类的
}

type GateUserInfo struct {
	userId uint64
	connId uint64
}

//*GateLogic继承于LogicProcess接口
type GateLogic struct {
	mPool          *PoolAndAgent.SingleMsgPool //自身绑定的SingleMsgPool
	mListenAgent   *PoolAndAgent.NetAgent
	mRouterAgents  *PoolAndAgent.RouterAgent //暂时一个router agent，以后可能会有多个
	mMyAppid       uint32
	mMapConnection map[uint64]GateConnection //以connId为key的map
	mMapUser       map[uint64]GateUserInfo   //以userId为key的map
}

//实现PoolAndAgent.ILogicProcess的三个接口函数
func (this *GateLogic) Init(myPool *PoolAndAgent.SingleMsgPool) bool {
	this.mPool = myPool
	this.mMapConnection = make(map[uint64]GateConnection)
	this.mMapUser = make(map[uint64]GateUserInfo)
	return true
}

func (this *GateLogic) ProcessReq(req proto.Message, pDatabase *PoolAndAgent.CADODatabase) {
	if req == nil {
		return
	}
	msg := Gate_CreateCommonMsgByTCPTransferMsg(req)
	switch data := msg.(type) {
	case *PrivateInitMsg:
		this.Private_OnInit(data)
	case *bs_tcp.TCPSessionCome:
		this.Network_OnConnOK(data)
	case *bs_tcp.TCPSessionClose:
		this.Network_OnConnClose(data)
	case *bs_gate.GateTransferData:
		this.Gate_GateTransferData(data)
	case *bs_gate.PulseReq:
		this.Gate_PulseReq(data)
	case *bs_router.RouterTransferData:
		this.Router_OnRouterTransferData(data)
	case *bs_client.LoginRsp:
		this.Client_OnLoginRsp(data)
	default:
		fmt.Println("不识别的报文，string=", data.String())
	}
}

func (this *GateLogic) OnPulse(ms uint64) {
	//定时调用程序
}

func (this *GateLogic) Private_OnInit(req *PrivateInitMsg) {
	this.mMyAppid = req.myAppId
}

//新建了一个客户端session
func (this *GateLogic) Network_OnConnOK(req *bs_tcp.TCPSessionCome) {
	fmt.Println("收到了TCPSessionCome报文")
	if req.Base.ConnId > 0xefffffff {
		fmt.Println("报告大王，连接快被分配完了，快点采取行动")
	}

	if _, ok := this.mMapConnection[req.Base.ConnId]; ok {
		//FIXME 一般这不可能发生
		fmt.Println("发生了不可能事件，有重复的connId发生，connId = ", req.Base.ConnId)
	} else {
		fmt.Println("新建了一个客户端连接，connId = ", req.Base.ConnId)
		unixNow := time.Now().Unix()
		this.mMapConnection[req.Base.ConnId] = GateConnection{
			connId:           req.Base.ConnId,
			clientAdress:     req.Base.RemoteAdd,
			userId:           0,
			isAuthenticated:  false,
			connectedTime:    unixNow,
			lastResponseTime: unixNow}
	}

}

//断开了一个客户端session
func (this *GateLogic) Network_OnConnClose(req *bs_tcp.TCPSessionClose) {
	connId := req.Base.ConnId
	fmt.Println("conn_id=", req.Base.ConnId, "断开连接")
	connElem, ok := this.mMapConnection[connId]
	if !ok {
		return
	}
	//	SendLogoutToOnline(connId) //这个函数暂时为空
	connElem.isConnectted = false
	userElem, ok2 := this.mMapUser[connElem.userId]
	if ok2 && userElem.connId == connId {
		//在conn_id相等的情况下才能干掉m_map_user里的对应user_id，不然的话就删错了有木有
		delete(this.mMapUser, connElem.userId)
	}
	delete(this.mMapConnection, connId)
}

//收到了客户端的心跳测试请求
func (this *GateLogic) Gate_PulseReq(req *bs_gate.PulseReq) {
	//发送回复
	rsp := new(bs_gate.PulseRsp)
	bs_proto.SetBaseKindAndSubId(rsp)
	rsp.Base.ConnId = req.Base.ConnId
	rsp.SpeedData = uint32(time.Now().Unix())
	this.SendToClient(rsp, rsp.Base)
}

//收到了客户端传来的消息
func (this *GateLogic) Gate_GateTransferData(req *bs_gate.GateTransferData) {
	connElem, ok := this.mMapConnection[req.Base.ConnId]
	if !ok {
		//FIXME logger
		fmt.Println("找不到对应connId=", req.Base.ConnId)
		return
	}

	connElem.msgFromClientNum++
	connElem.lastResponseTime = time.Now().Unix()

	routerMsg := new(bs_router.RouterTransferData)
	bs_proto.SetBaseKindAndSubId(routerMsg)
	bs_proto.CopyBaseExceptKindAndSubId(routerMsg.Base, req.Base)

	routerMsg.DestAppid = req.AttAppid
	routerMsg.DestApptype = req.AttApptype
	routerMsg.DataCmdKind = req.DataCmdKind
	routerMsg.DataCmdSubid = req.DataCmdSubid
	routerMsg.Data = make([]byte, len(req.Data))
	copy(routerMsg.Data, req.Data) //这里使用copy重新分配一块内存块，然后req就可以被GC了，可能routerMsg.Data = req.Data更好，可以跑跑试试
	//FIXME 这里需要取得自己的app_type与id
	routerMsg.SrcAppid = this.mMyAppid
	routerMsg.SrcApptype = uint32(bs_types.EnumAppType_Gate)
	routerMsg.DataDirection = bs_router.RouterTransferData_Client2App
	routerMsg.AttGateid = this.mMyAppid
	routerMsg.AttUserid = connElem.userId
	routerMsg.AttGateconnid = connElem.connId
	routerMsg.ClientRemoteAddress = connElem.clientAdress

	if routerMsg.DestAppid == 0 {
		fmt.Println("目标地址为0,来自client,connid=", connElem.connId,
			",kind_id=", req.DataCmdKind,
			",sub_id=", req.DataCmdSubid,
			",my_user_id=", connElem.userId,
			",dest_appid=", req.AttAppid,
			",dest_apptype=", bs_proto.GetAppTypeName(req.AttApptype))
		return //且不往router发送了
	}
	//向router发送消息
	fmt.Println("向router发送客户端消息")
	this.mPool.SendMsgToServerAppByRouter(routerMsg)
}

func (this *GateLogic) Router_OnRouterTransferData(req *bs_router.RouterTransferData) {

	//转发报文,应当只会是那类需要转到客户端的
	//不是转发到客户端的报文，在解析成普通报文后重新推送进自己的mPool中，是否要处理，在processReq入口处决定
	//

	//这里发送对象以userId为准，如果userId为0才去使用gateconnid来查找相应发送对象
	gateConnId := req.AttGateconnid
	if req.DataDirection == bs_router.RouterTransferData_App2Client {
		userId := req.AttUserid
		if userId == 0 && gateConnId == 0 {
			fmt.Println("报告大王，有个搞笑的家伙发来了user_id和conn_id均为0的报文。叫我女王大人。好的大王，没问题大王！")
			fmt.Println("这个家伙是, 来自router,connid=", req.AttGateconnid,
				",kind_id=", req.DataCmdKind,
				",sub_id=", req.DataCmdSubid,
				",att_user_id=", req.AttUserid,
				",src_appid=", req.SrcAppid,
				",src_apptype=", req.SrcApptype)
			//可能需要通知写这个APP的程序猿
			return
		}

		if userId != 0 {
			//如果user_id不为0，则发送对象以user_id为准，因为gate_connid在顶号登录时，其他APP没有同步更新得到新的conn_id，这样就会把报文发给已经断线的客户端
			if userElem, ok := this.mMapUser[userId]; !ok {
				//如果用户不为0,又在mMapUser找不到对应用户，说明用户已断线离开,直接return
				fmt.Println("用户已找不到，无法将该报文发往指定用户，可能该用户已经掉线，user_id=", userId)
				return
			} else {
				//对gateConnId重新赋值，当userId不为0时，以userId查map所得的connId为准
				gateConnId = userElem.connId
			}
		}

		connElem, ok := this.mMapConnection[gateConnId]
		if !ok {
			fmt.Println("找不到相应connId, 来自router,connid=", req.AttGateconnid,
				",kind_id=", req.DataCmdKind,
				",sub_id=", req.DataCmdSubid,
				",my_user_id=", connElem.userId,
				",att_user_id=", req.AttUserid,
				",src_appid=", req.SrcAppid,
				",src_apptype=", req.SrcApptype)
		}

		if req.AttUserid != 0 && connElem.isAuthenticated != true {
			//若已经成功登录以后的
			fmt.Println("该连接的用户尚未认证，但是却收到了发往该用户的报文，user_id=", req.AttUserid)
			return
		} else {
			if gateConnId != req.AttGateconnid {
				//就是在if userElem, ok := this.mMapUser[userId]; !ok被重新赋值了
				fmt.Println("报文的gateconnid与gate自己记录的connid不匹配，可能该账号是顶号登录的",
					",报文connid=", req.AttGateconnid,
					",实际connid=", gateConnId,
					",user_id=", userId,
					",src_appid=", req.SrcAppid)
			}
			req.AttGateconnid = gateConnId
			this.SendToClient(req, req.Base)
		}

	} else if req.DataDirection == bs_router.RouterTransferData_App2App {
		//这个是发给gate自己的，而不是发给客户端的，需要gate app来处理
		fmt.Println("收到了需要gate自己来处理的报文,SrcApptype=", req.SrcApptype, "SrcAppid=", req.SrcAppid,
			"DataCmdKind=", req.DataCmdKind, "DataCmdSubid=", req.DataCmdSubid)
		msg := Gate_CreateCommonMsgByRouterTransferMsg(req)
		if msg != nil {
			this.mPool.PushMsg(msg, 0) //推送到pool的chan队列尾
		}
	} else { //clientToApp,这是不可能的
		//from client?
		//FIXME log
		//you should kick this connection
	}

}

func (this *GateLogic) AppFrame_OnClientAuth(req proto.Message) {

}

//登录回复报文
func (this *GateLogic) Client_OnLoginRsp(req *bs_client.LoginRsp) {
	fmt.Println("收到了LoginRsp, string=", req.String(), "gate connId=", req.Base.GateConnId)
	//收到了登录回复报文表示验证成功了
	connId := req.Base.GateConnId

	userId := req.UserId
	connElem, ok := this.mMapConnection[connId]
	if !ok {
		bs_proto.OutputMyLog("不存在的connId=", connId)
		return
	}
	connElem.isAuthenticated = true
	connElem.userId = userId
	this.mMapConnection[connId] = connElem
	//登录成功的情况下才记录userId
	if userId != 0 && req.LoginResult == bs_client.LoginRsp_SUCCESS {
		//查mMapUser
		userElem, ok := this.mMapUser[userId]
		if ok {
			//如果存在，要做顶号处理，把之前的连接断开，不能一个userId维持两个连接
			kick := new(bs_tcp.TCPSessionKick) //userElem.connId就是要断开的connId
			kick.Base.ConnId = userElem.connId
			this.SendToClient(kick, kick.Base)
		}
		userElem.userId = userId
		userElem.connId = connId
		this.mMapUser[userId] = userElem
	}
	req.Base.ConnId = connId
	fmt.Println("向client发送,将Base.ConnId重置后发送")
	this.SendToClient(req, req.Base)
}

//向客户端发送报文
func (this *GateLogic) SendToClient(req proto.Message, pBase *bs_types.BaseInfo) {
	//先把其他报文转成bs_gate.TransferData然后再转成bs_tcp.TCPTransferMsg
	var gateTrans *bs_gate.GateTransferData = nil
	gateTrans = Gate_CreateGateTransferMsgByCommonMsg(req)
	if gateTrans != nil {
		//将bs_gate.TransferData报文转为bs_tcp.TCPTransferMsg
		msg := Gate_CreateTCPTransferMsgByCommonMsg(gateTrans, gateTrans.Base)
		this.mPool.SendMsgToClientByNetAgent(msg)
	} else {
		//非bs_gate.TransferData报文就直接把req转为bs_tcp.TCPTransferMsg
		msg := Gate_CreateTCPTransferMsgByCommonMsg(req, pBase)
		this.mPool.SendMsgToClientByNetAgent(msg)
	}

}

//主动断开一个session连接
func (this *GateLogic) CloseSession(connId uint64) {
	kick := new(bs_tcp.TCPSessionKick)
	bs_proto.SetBaseKindAndSubId(kick)
	kick.Base.ConnId = connId
	//通知tcpnet层断开这个session
	this.mPool.SendMsgToClientByNetAgent(kick)
}
