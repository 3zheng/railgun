package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/3zheng/railcommon"
	protodf "github.com/3zheng/railproto"

	proto "google.golang.org/protobuf/proto"
)

type RouterConnection struct {
	connId           uint64
	isConnectted     bool
	clientAdress     string
	appId            uint32 //对应的APP ID
	appType          uint32 //对应的app type
	lastResponseTime int64  //time.Now().Unix()，最近一次的响应时间，用于判断是否因为超时需要断开这个链接
}

type RouterLogic struct {
	mPool        *railcommon.SingleMsgPool //自身绑定的SingleMsgPool
	mListenAgent *railcommon.NetAgent
	mMyAppid     uint32
	//有分为app id和app type两个map是因为:有时候发送消息并不需要指定appid，只需要发往实现这个功能的apptype就行了
	//又因为注册app和断开app都是低频事件，而报文发送是高频事件，所以mMapAppType的value使用的是slice而不是map[uint32]uint32
	mMapConnection map[uint64]uint32           //以conn id为key, app id为value的map
	mMapAppId      map[uint32]RouterConnection //以app id为key的map
	mMapAppType    map[uint32]([]uint32)       //以app type为key, 同一个app type下的所有app id组成的数组为value的map
	mRandNum       *rand.Rand                  //随机数，在send anyone app的时候用到
}

func (this *RouterLogic) Init(myPool *railcommon.SingleMsgPool) bool {
	this.mPool = myPool
	//创建以UnixNano为随机种子的随机数变量
	s2 := rand.NewSource(time.Now().UnixNano())
	this.mRandNum = rand.New(s2)
	this.mMapConnection = make(map[uint64]uint32)
	this.mMapAppId = make(map[uint32]RouterConnection)
	this.mMapAppType = make(map[uint32]([]uint32))
	return true
}

func (this *RouterLogic) ProcessReq(req proto.Message, pDatabase *railcommon.CADODatabase) {
	msg := Router_CreateCommonMsgByTCPTransferMsg(req)
	switch data := msg.(type) {
	case *PrivateInitMsg:
		this.Private_OnInit(data)
	case *protodf.TCPSessionCome:
		this.Network_OnConnOK(data)
	case *protodf.TCPSessionClose:
		this.Network_OnConnClose(data)
	case *protodf.RouterTransferData:
		this.Router_OnRouteDataReq(data)
	case *protodf.RegisterAppReq:
		this.Router_OnRegReq(data)
	default:
		return
	}
}

func (this *RouterLogic) OnPulse(ms uint64) {

}

func (this *RouterLogic) Private_OnInit(req *PrivateInitMsg) {
	this.mMyAppid = req.myAppId
}

func (this *RouterLogic) Router_OnRegReq(req *protodf.RegisterAppReq) {
	appId := req.AppId
	appType := req.AppType
	connId := req.Base.ConnId
	if _, ok1 := this.mMapConnection[connId]; !ok1 {
		//如果ok1不存在返回
		return
	}
	_, ok2 := this.mMapAppId[appId]
	if ok2 {
		fmt.Println("相同类型相同app_id的app已经注册了，不允许重复注册。apptype=", appType, "typename=", protodf.GetAppTypeName(appType),
			",appid=", appId, ",connid=", req.Base.ConnId)
		log.Println("相同类型相同app_id的app已经注册了，不允许重复注册。apptype=", appType, "typename=", protodf.GetAppTypeName(appType),
			",appid=", appId, ",connid=", req.Base.ConnId)
		this.CloseSession(req.Base.ConnId)
		return
	}
	//发送回复
	rsp := new(protodf.RegisterAppRsp)
	protodf.SetBaseKindAndSubId(rsp)
	protodf.CopyBaseExceptKindAndSubId(rsp.Base, req.Base)
	rsp.RegResult = 1
	this.SendToOtherApp(rsp, rsp.Base)

	//往mMapConnection，mMapAppId，mMapAppType里新增数据
	this.mMapConnection[connId] = appId
	this.mMapAppId[appId] = RouterConnection{
		connId:           connId,
		isConnectted:     true,
		clientAdress:     req.Base.RemoteAdd,
		appId:            appId,
		appType:          appType,
		lastResponseTime: time.Now().Unix()}

	typeElem, ok3 := this.mMapAppType[appType]
	if !ok3 {
		sliceAppId := make([]uint32, 0, 10)
		this.mMapAppType[appType] = sliceAppId
		typeElem, ok3 = this.mMapAppType[appType]
	}
	var exist bool = false
	for _, v := range typeElem {
		//查找相应appid是否已经存在
		if v == appId {
			exist = true
			break
		}
	}
	if exist {
		//对应的appId存在
		fmt.Println(fmt.Println("奇怪的错误，相同类型相同app_id的app在mMapAppType中找到了，",
			"但是却通过了前面的appid map和connid map的检查。apptype=", appType,
			",appid=", appId, ",connid=", req.Base.ConnId))
	} else {
		//对应的appId不存在,新增这个appId
		typeElem = append(typeElem, appId)
		this.mMapAppType[appType] = typeElem
		fmt.Println("mMapAppType=", this.mMapAppType)
	}

	fmt.Println("一个App注册来了,type=", appType,
		",typename=", protodf.GetAppTypeName(appType),
		",id=", appId)

}

func (this *RouterLogic) Router_OnRouteDataReq(req *protodf.RouterTransferData) {
	//----------------------------------------------------------------------
	// 转发报文
	// 目前看起来，指定app与指定appType中的任何一个使用的较多
	//----------------------------------------------------------------------

	connId := req.Base.ConnId
	destAppId := req.DestAppid
	destAppType := req.DestApptype
	//判断connid是否存在，
	if appId, ok1 := this.mMapConnection[connId]; !ok1 {
		fmt.Println("当前连接还没有注册，不转发其报文:connid=", connId)
		return
	} else {
		elem, ok2 := this.mMapAppId[appId]
		if ok2 {
			//更新最新响应时间
			elem.lastResponseTime = time.Now().Unix()
			this.mMapAppId[appId] = elem
			//根据从mMapAppId给req的srcApptype和srcAppId赋值,这个值以router为准,不以传过来的报文为准
			req.SrcApptype = elem.appType
			req.SrcAppid = elem.appId
		}
	}

	var sendResult bool = false
	switch destAppId {
	case uint32(protodf.EnumAppId_Send2AnyOne):
		//发给某种apptype下的任意一个appid
		sendResult = this.DeliverToAnyOneByType(req, destAppType)
	case uint32(protodf.EnumAppId_Send2All):
		//发给某个apptype下的所有appid
		sendResult = this.DeliverToAllByType(req, destAppType)
	default:
		sendResult = this.DeliverToAllByPointID(req, destAppType, destAppId)
	}
	if !sendResult {
		fmt.Println("目标APP无法找到，可能尚未注册",
			",dest apptype=", req.DestApptype,
			",dest appname=", protodf.GetAppTypeName(req.DestApptype),
			",dest appid=", req.DestAppid,
			",src apptype=", req.SrcApptype,
			",src appname=", protodf.GetAppTypeName(req.SrcApptype),
			",src appid=", req.SrcAppid,
			",src connid=", req.Base.ConnId,
			",cmd_kindid=", req.DataCmdKind,
			",cmd_subid=", req.DataCmdSubid,
			",userid=", req.AttUserid,
			",gate connid=", req.AttGateconnid)
	} else {
		fmt.Println("已向目标APP发送报文",
			",dest apptype=", req.DestApptype,
			",dest appname=", protodf.GetAppTypeName(req.DestApptype),
			",dest appid=", req.DestAppid,
			",src apptype=", req.SrcApptype,
			",src appname=", protodf.GetAppTypeName(req.SrcApptype),
			",src appid=", req.SrcAppid,
			",src connid=", req.Base.ConnId,
			",cmd_kindid=", req.DataCmdKind,
			",cmd_subid=", req.DataCmdSubid,
			",userid=", req.AttUserid,
			",gate connid=", req.AttGateconnid)
	}
}

func (this *RouterLogic) Network_OnConnOK(req *protodf.TCPSessionCome) {
	if _, ok := this.mMapConnection[req.Base.ConnId]; ok {
		//FIXME 一般这不可能发生
		fmt.Println("发生了不可能事件，有重复的connId发生，connId = ", req.Base.ConnId)
	} else {
		fmt.Println("新建了一个客户端连接，connId = ", req.Base.ConnId)
		this.mMapConnection[req.Base.ConnId] = 0 //直到收到regreq后才知道app id，先设为0
	}
}

func (this *RouterLogic) Network_OnConnClose(req *protodf.TCPSessionClose) {
	connId := req.Base.ConnId
	fmt.Println("conn_id=", req.Base.ConnId, "断开连接")
	appId, ok := this.mMapConnection[connId]
	if !ok {
		return
	}
	//	SendLogoutToOnline(connId) //这个函数暂时为空
	idElem, ok2 := this.mMapAppId[appId]
	if ok2 && idElem.connId == connId {
		//找到mMapAppType里对应的app id
		if typeElem, ok3 := this.mMapAppType[idElem.appType]; ok3 {
			for i, v := range typeElem {
				if v == appId {
					//将v这个元素从typeElem里删除
					typeElem = append(typeElem[:i], typeElem[i+1:]...)
					this.mMapAppType[idElem.appType] = typeElem
				}
			}
		}
		//在conn_id相等的情况下才能干掉mMapAppId里的对应appId，不然的话就删错了有木有
		delete(this.mMapAppId, appId)
	}
	delete(this.mMapConnection, connId)
}

// 向客户端发送报文
func (this *RouterLogic) SendToOtherApp(req proto.Message, pBase *protodf.BaseInfo) {
	msg := Router_CreateTCPTransferMsgByCommonMsg(req, pBase)
	this.mPool.SendMsgToClientByNetAgent(msg)
}

// 主动断开一个session连接
func (this *RouterLogic) CloseSession(connId uint64) {
	kick := new(protodf.TCPSessionKick)
	protodf.SetBaseKindAndSubId(kick)
	kick.Base.ConnId = connId
	//通知tcpnet层断开这个session
	this.mPool.SendMsgToClientByNetAgent(kick)
}

// 向某种apptype下的任意一个appid发送报文
func (this *RouterLogic) DeliverToAnyOneByType(req *protodf.RouterTransferData, destAppType uint32) bool {
	protodf.OutputMyLog("发送往任意的", protodf.GetAppTypeName(destAppType))
	if req.DataDirection == protodf.RouterTransferData_App2Client {
		//如果是服务端发往客户端的报文。不接受发往任意gate，必须指定gate appid，因为发往任意gate没有意义，根本找不到对应用户
		protodf.OutputMyLog("服务端发往客户端的报文必须指定gate appid")
		return false
	}
	typeElem, ok2 := this.mMapAppType[destAppType]
	if !ok2 {
		protodf.OutputMyLog("找不到相应的AppType=", protodf.GetAppTypeName(destAppType))
		return false
	}
	//先找到对应的app type然后随机选择属于这个app type的一个app id
	if size := len(typeElem); size > 0 {
		randIndex := this.mRandNum.Intn(size) //返回的是随机数对size求余后的范围在[0,size)的新随机数
		randAppId := typeElem[randIndex]
		idElem, ok3 := this.mMapAppId[randAppId]
		if !ok3 {
			protodf.OutputMyLog("找不到相应的AppId=", randAppId)
			return false
		}
		//修改connId然后就可以转发了
		req.Base.ConnId = idElem.connId
		this.SendToOtherApp(req, req.Base)
	} else {
		return false
	}
	return true
}

// 向某种apptype下的所有appid发送报文
func (this *RouterLogic) DeliverToAllByType(req *protodf.RouterTransferData, destAppType uint32) bool {
	if req.DataDirection == protodf.RouterTransferData_App2Client && destAppType == uint32(protodf.EnumAppType_Gate) {
		//如果是服务端发往客户端的报文。接受发往所有gate，因为有可能是广播类型消息，但是要慎用，所以打印出来，以免滥用
		fmt.Println("此报文向将所有gate广播",
			",dest apptype=", req.DestApptype,
			",dest appname=", protodf.GetAppTypeName(req.DestApptype),
			",dest appid=", req.DestAppid,
			",src apptype=", req.SrcApptype,
			",src appname=", protodf.GetAppTypeName(req.SrcApptype),
			",src appid=", req.SrcAppid,
			",src connid=", req.Base.ConnId,
			",cmd_kindid=", req.DataCmdKind,
			",cmd_subid=", req.DataCmdSubid,
			",userid=", req.AttUserid,
			",gate connid=", req.AttGateconnid)
	}

	typeElem, ok2 := this.mMapAppType[destAppType]
	if !ok2 {
		return false
	}
	//遍历typeElem并向每一个appid发送
	for _, appId := range typeElem {
		idElem, ok3 := this.mMapAppId[appId]
		if !ok3 {
			return false
		}
		//修改connId然后就可以转发了
		req.Base.ConnId = idElem.connId
		this.SendToOtherApp(req, req.Base)
	}
	return true
}

// 向指定appId发送报文
func (this *RouterLogic) DeliverToAllByPointID(req *protodf.RouterTransferData, destAppType uint32, destAppId uint32) bool {
	if req.DataDirection == protodf.RouterTransferData_App2Client && destAppType != uint32(protodf.EnumAppType_Gate) {
		//如果是服务端发往客户端的报文。却不是发往gate，那就是填错了
		return false
	}

	idElem, ok3 := this.mMapAppId[destAppId]
	if !ok3 {
		return false
	}
	//验证报文的destAppType是否正确
	if idElem.appType != destAppType {
		return false
	}
	//修改connId然后就可以转发了
	req.Base.ConnId = idElem.connId
	this.SendToOtherApp(req, req.Base)
	return true
}
