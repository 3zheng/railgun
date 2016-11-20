package bs_proto

import (
	"fmt"

	bs_client "railgun/protodefine/client"
	bs_gate "railgun/protodefine/gate"
	bs_types "railgun/protodefine/mytype"
	bs_router "railgun/protodefine/router"
	bs_tcp "railgun/protodefine/tcpnet"
)

//设置input的baseinfo值，如果返回false说明这个类型找不到
func SetBaseKindAndSubId(input interface{}) (bool, *bs_types.BaseInfo) {
	if input == nil {
		return false, nil
	}
	switch data := input.(type) {
	//以下报文属于tcp.proto，CMDKindId_IDKindNetTCP大类
	case *bs_tcp.TCPTransferMsg:
		fmt.Println("报文类型为*bs_tcp.TCPTransferMsg")
		if data.Base == nil {
			fmt.Println("分配了new(bs_types.BaseInfo)")
			data.Base = new(bs_types.BaseInfo)
		}
		data.Base.KindId = uint32(bs_types.CMDKindId_IDKindNetTCP)
		data.Base.SubId = uint32(bs_tcp.CMDID_Tcp_IDTCPTransferMsg)
		return true, data.Base
	case *bs_tcp.TCPSessionCome:
		if data.Base == nil {
			data.Base = new(bs_types.BaseInfo)
		}
		data.Base.KindId = uint32(bs_types.CMDKindId_IDKindNetTCP)
		data.Base.SubId = uint32(bs_tcp.CMDID_Tcp_IDTCPSessionCome)
		return true, data.Base
	case *bs_tcp.TCPSessionClose:
		if data.Base == nil {
			data.Base = new(bs_types.BaseInfo)
		}
		data.Base.KindId = uint32(bs_types.CMDKindId_IDKindNetTCP)
		data.Base.SubId = uint32(bs_tcp.CMDID_Tcp_IDTCPSessionClose)
		return true, data.Base
	case *bs_tcp.TCPSessionKick:
		if (*data).Base == nil {
			(*data).Base = new(bs_types.BaseInfo)
		}
		(*data).Base.KindId = uint32(bs_types.CMDKindId_IDKindNetTCP)
		(*data).Base.SubId = uint32(bs_tcp.CMDID_Tcp_IDTCPSessionKick)
		return true, data.Base
	//以下报文属于gate.proto,CMDKindId_IDKindGate大类
	case *bs_gate.PulseReq:
		if data.Base == nil {
			data.Base = new(bs_types.BaseInfo)
		}
		data.Base.KindId = uint32(bs_types.CMDKindId_IDKindGate)
		data.Base.SubId = uint32(bs_gate.CMDID_Gate_IDPulseReq)
		return true, data.Base
	case *bs_gate.PulseRsp:
		if data.Base == nil {
			data.Base = new(bs_types.BaseInfo)
		}
		data.Base.KindId = uint32(bs_types.CMDKindId_IDKindGate)
		data.Base.SubId = uint32(bs_gate.CMDID_Gate_IDPulseRsp)
		return true, data.Base
	case *bs_gate.TransferData:
		if data.Base == nil {
			data.Base = new(bs_types.BaseInfo)
		}
		data.Base.KindId = uint32(bs_types.CMDKindId_IDKindGate)
		data.Base.SubId = uint32(bs_gate.CMDID_Gate_IDTransferData)
		return true, data.Base
	//以下报文属于router.proto,CMDKindId_IDKindRouter大类
	case *bs_router.RouterTransferData:
		if data.Base == nil {
			data.Base = new(bs_types.BaseInfo)
		}
		data.Base.KindId = uint32(bs_types.CMDKindId_IDKindRouter)
		data.Base.SubId = uint32(bs_router.CMDID_Router_IDTransferData)
		return true, data.Base
	case *bs_router.RegisterAppReq:
		if data.Base == nil {
			data.Base = new(bs_types.BaseInfo)
		}
		data.Base.KindId = uint32(bs_types.CMDKindId_IDKindRouter)
		data.Base.SubId = uint32(bs_router.CMDID_Router_IDRegisterAppReq)
		return true, data.Base
	case *bs_router.RegisterAppRsp:
		if data.Base == nil {
			data.Base = new(bs_types.BaseInfo)
		}
		data.Base.KindId = uint32(bs_types.CMDKindId_IDKindRouter)
		data.Base.SubId = uint32(bs_router.CMDID_Router_IDRegisterAppRsp)
		return true, data.Base
	//以下报文属于client.proto,CMDKindId_IDKindClient大类
	case *bs_client.LoginReq:
		if data.Base == nil {
			data.Base = new(bs_types.BaseInfo)
		}
		data.Base.KindId = uint32(bs_types.CMDKindId_IDKindClient)
		data.Base.SubId = uint32(bs_client.CMDID_Client_IDLoginReq)
		return true, data.Base
	case *bs_client.LoginRsp:
		if data.Base == nil {
			data.Base = new(bs_types.BaseInfo)
		}
		data.Base.KindId = uint32(bs_types.CMDKindId_IDKindClient)
		data.Base.SubId = uint32(bs_client.CMDID_Client_IDLoginRsp)
		if data.UserBaseInfo == nil { //顺便设置一下复合数据类型
			data.UserBaseInfo = new(bs_types.BaseUserInfo)
		}
		if data.UserSesionInfo == nil {
			data.UserSesionInfo = new(bs_types.UserSessionInfo)
		}
		return true, data.Base
	case *bs_client.LogoutReq:
		if data.Base == nil {
			data.Base = new(bs_types.BaseInfo)
		}
		data.Base.KindId = uint32(bs_types.CMDKindId_IDKindClient)
		data.Base.SubId = uint32(bs_client.CMDID_Client_IDLogoutReq)
		return true, data.Base
	case *bs_client.LogoutRsp:
		if data.Base == nil {
			data.Base = new(bs_types.BaseInfo)
		}
		data.Base.KindId = uint32(bs_types.CMDKindId_IDKindClient)
		data.Base.SubId = uint32(bs_client.CMDID_Client_IDLogoutRsp)
		return true, data.Base
	case *bs_client.QueryFundReq:
		if data.Base == nil {
			data.Base = new(bs_types.BaseInfo)
		}
		data.Base.KindId = uint32(bs_types.CMDKindId_IDKindClient)
		data.Base.SubId = uint32(bs_client.CMDID_Client_IDQueryFundReq)
		return true, data.Base
	case *bs_client.QueryFundRsp:
		if data.Base == nil {
			data.Base = new(bs_types.BaseInfo)
		}
		data.Base.KindId = uint32(bs_types.CMDKindId_IDKindClient)
		data.Base.SubId = uint32(bs_client.CMDID_Client_IDQueryFundRsp)
		return true, data.Base
	case *bs_client.GetOnlineUserReq:
		if data.Base == nil {
			data.Base = new(bs_types.BaseInfo)
		}
		data.Base.KindId = uint32(bs_types.CMDKindId_IDKindClient)
		data.Base.SubId = uint32(bs_client.CMDID_Client_IDGetOnlineUserReq)
		return true, data.Base
	case *bs_client.GetOnlineUserRsp:
		if data.Base == nil {
			data.Base = new(bs_types.BaseInfo)
		}
		data.Base.KindId = uint32(bs_types.CMDKindId_IDKindClient)
		data.Base.SubId = uint32(bs_client.CMDID_Client_IDGetOnlineUserRsp)
		return true, data.Base
	case *bs_client.KickUserReq:
		if data.Base == nil {
			data.Base = new(bs_types.BaseInfo)
		}
		data.Base.KindId = uint32(bs_types.CMDKindId_IDKindClient)
		data.Base.SubId = uint32(bs_client.CMDID_Client_IDKickUserReq)
		return true, data.Base
	case *bs_client.KickUserRsp:
		if data.Base == nil {
			data.Base = new(bs_types.BaseInfo)
		}
		data.Base.KindId = uint32(bs_types.CMDKindId_IDKindClient)
		data.Base.SubId = uint32(bs_client.CMDID_Client_IDKickUserRsp)
		return true, data.Base
	default:
		fmt.Println("input为不识别的类型")
		return false, nil
	}
	return false, nil
}

//复制除了kindid和subid以外的值
func CopyBaseExceptKindAndSubId(dst *bs_types.BaseInfo, src *bs_types.BaseInfo) {
	if dst == nil || src == nil {
		fmt.Println("CopyBaseExceptKindAndSubId传入的参数是空,dst=", dst, ",src=", src)
		return
	}
	dst.ConnId = src.ConnId
	dst.GateConnId = src.GateConnId
	dst.RemoteAdd = src.RemoteAdd
	dst.AttApptype = src.AttApptype
	dst.AttAppid = src.AttAppid
}

func SetCommonMsgBaseByRouterTransferData(dst *bs_types.BaseInfo, srcRouter *bs_router.RouterTransferData) {
	if dst == nil || srcRouter == nil {
		fmt.Println("SetCommonMsgBaseByRouterTransferData传入的参数是空,dst=", dst, ",src=", srcRouter)
		return
	}
	CopyBaseExceptKindAndSubId(dst, srcRouter.Base)
	//和客户端相关的都要重新赋值，取的是srcRouter的值，而不是srcRouter.Base的值
	dst.AttAppid = srcRouter.SrcAppid
	dst.AttApptype = srcRouter.SrcApptype
	dst.RemoteAdd = srcRouter.ClientRemoteAddress
	dst.GateConnId = srcRouter.AttGateconnid
}
