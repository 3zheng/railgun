package main

import (
	"fmt"

	bs_proto "github.com/3zheng/railgun/protodefine"
	bs_client "github.com/3zheng/railgun/protodefine/client"
	bs_gate "github.com/3zheng/railgun/protodefine/gate"
	bs_types "github.com/3zheng/railgun/protodefine/mytype"
	bs_router "github.com/3zheng/railgun/protodefine/router"
	bs_tcp "github.com/3zheng/railgun/protodefine/tcpnet"
	proto "google.golang.org/protobuf/proto"
)

//gate消息过滤器,用于一般消息和bs_tcp.TCPTransferMsg消息的相互转换

// 这个函数处理来自pool的消息，因为pool里面的消息不止来源于net层，也可能来源于其他逻辑层或者自身的延时发送消息
// 所以，这个函数里的消息不止有TCPTransferMsg，但是只需要处理TCPTransferMsg这个消息就行了
// 把从TCPTransferMsg解析出来的gate应该处理的报文返回，不处理的报文直接丢弃返回nil
func Gate_CreateCommonMsgByTCPTransferMsg(req proto.Message) proto.Message {
	switch data := req.(type) {
	case *bs_tcp.TCPTransferMsg:
		fmt.Println("bs_tcp.TCPTransferMsg base=", data.Base)
		switch data.DataKindId { //判断大类
		case uint32(bs_types.CMDKindId_IDKindGate):
			switch data.DataSubId { //判断小类
			case uint32(bs_gate.CMDID_Gate_IDPulseReq):
				msg := new(bs_gate.PulseReq)
				err := proto.Unmarshal(data.Data, msg)
				bs_proto.SetBaseKindAndSubId(msg)
				bs_proto.CopyBaseExceptKindAndSubId(msg.Base, data.Base)
				if err != nil {
					fmt.Println("解析PulseReq出错")
					return nil
				}
				return msg
			case uint32(bs_gate.CMDID_Gate_IDPulseRsp):
				msg := new(bs_gate.PulseRsp)
				err := proto.Unmarshal(data.Data, msg)
				bs_proto.SetBaseKindAndSubId(msg)
				bs_proto.CopyBaseExceptKindAndSubId(msg.Base, data.Base)
				if err != nil {
					fmt.Println("解析PulseRsp出错")
					return nil
				}
				return msg
			case uint32(bs_gate.CMDID_Gate_IDTransferData):
				msg := new(bs_gate.GateTransferData)
				err := proto.Unmarshal(data.Data, msg)
				bs_proto.SetBaseKindAndSubId(msg)
				bs_proto.CopyBaseExceptKindAndSubId(msg.Base, data.Base)
				if err != nil {
					fmt.Println("解析TransferData出错")
					return nil
				}
				fmt.Println("成功解析TransferData")
				return msg
			default:
				fmt.Println("不识别的gate报文，DataSubId=", data.DataSubId)
				return nil //丢弃这个TCPTransferMsg报文
			}
		case uint32(bs_types.CMDKindId_IDKindRouter):
			switch data.DataSubId { //判断小类
			case uint32(bs_router.CMDID_Router_IDTransferData):
				msg := new(bs_router.RouterTransferData)
				err := proto.Unmarshal(data.Data, msg)
				bs_proto.SetBaseKindAndSubId(msg)
				bs_proto.CopyBaseExceptKindAndSubId(msg.Base, data.Base)
				if err != nil {
					fmt.Println("解析RouterTransferData出错")
					return nil
				}
				return msg
			default:
				fmt.Println("不识别的router报文，DataSubId=", data.DataSubId)
				return nil //丢弃这个TCPTransferMsg报文
			}
		default: //bs_tcp.TCPTransferMsg报文的成员变量大类DataKindId不为CMDKindId_IDKindGate和CMDKindId_IDKindRouter都丢弃
			return nil //丢弃这个TCPTransferMsg报文
		}
	default: //*bs_tcp.TCPTransferMsg以外的报文直接返回
		return req
	}
	return nil
}

// 这个函数处理发往pool的消息，因为除了普通消息以外还有kick掉一个session这样的消息
// 所以，这个函数里的要把kick消息区分出来，其他消息则转化成TCPTransferMsg
func Gate_CreateTCPTransferMsgByCommonMsg(req proto.Message, pBase *bs_types.BaseInfo) proto.Message {
	switch data := req.(type) {
	case *bs_tcp.TCPSessionKick:
		return data
	case *bs_tcp.TCPTransferMsg:
		return data
	default:
		msg := new(bs_tcp.TCPTransferMsg)
		bs_proto.SetBaseKindAndSubId(msg)
		bs_proto.CopyBaseExceptKindAndSubId(msg.Base, pBase)
		msg.DataKindId = pBase.KindId
		msg.DataSubId = pBase.SubId
		buf, err := proto.Marshal(req)
		if err != nil {
			return nil
		} else {
			msg.Data = buf
			return msg
		}
	}
	return req
}

func Gate_CreateCommonMsgByRouterTransferMsg(req *bs_router.RouterTransferData) proto.Message {
	if req == nil {
		return nil
	}

	//大类
	switch req.DataCmdKind {
	case uint32(bs_types.CMDKindId_IDKindClient):
		//小类
		switch req.DataCmdSubid {
		case uint32(bs_client.CMDID_Client_IDLoginRsp):
			msg := new(bs_client.LoginRsp)
			err := proto.Unmarshal(req.Data, msg)
			if err != nil {
				fmt.Println("LoginRsp解析失败")
				return nil
			}
			bs_proto.SetBaseKindAndSubId(msg)
			bs_proto.SetCommonMsgBaseByRouterTransferData(msg.Base, req)
			return msg
		default:
			return nil
		}
	default:
		return nil
	}

	return nil
}

func Gate_CreateGateTransferMsgByCommonMsg(req proto.Message) *bs_gate.GateTransferData {
	gateTrans := new(bs_gate.GateTransferData)
	bs_proto.SetBaseKindAndSubId(gateTrans)

	switch data := req.(type) {
	case *bs_router.RouterTransferData:
		//RouterTransferData要特殊对待，不是调用proto.Marshal
		bs_proto.CopyBaseExceptKindAndSubId(gateTrans.Base, data.Base)
		gateTrans.Base.ConnId = data.AttGateconnid
		gateTrans.DataCmdKind = data.DataCmdKind
		gateTrans.DataCmdSubid = data.DataCmdSubid
		gateTrans.Data = make([]byte, len(data.Data))
		copy(gateTrans.Data, data.Data) //使用copy，让req可以被GateConnection
		gateTrans.AttAppid = data.SrcAppid
		gateTrans.AttApptype = data.SrcApptype
		gateTrans.ReqId = 0
	case *bs_client.LoginRsp:
		fmt.Println("序列化bs_client.LoginRsp, LoginRsp=", data)
		bs_proto.CopyBaseExceptKindAndSubId(gateTrans.Base, data.Base)
		buff, err := proto.Marshal(data)
		if err != nil {
			fmt.Println("LoginRsp序列化失败")
			return nil
		}
		gateTrans.Data = buff
		gateTrans.DataCmdKind = uint32(bs_types.CMDKindId_IDKindClient)
		gateTrans.DataCmdSubid = uint32(bs_client.CMDID_Client_IDLoginRsp)
		gateTrans.AttAppid = data.Base.AttAppid
		gateTrans.AttApptype = data.Base.AttApptype
		gateTrans.ClientRemoteAddress = data.Base.RemoteAdd
		fmt.Println("LoginRsp => gateTrans=", gateTrans)
	default:
		return nil
	}
	return gateTrans
}
