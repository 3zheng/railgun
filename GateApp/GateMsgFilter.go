package main

import (
	"fmt"

	protodf "github.com/3zheng/railproto"
	proto "google.golang.org/protobuf/proto"
)

//gate消息过滤器,用于一般消息和protodf.TCPTransferMsg消息的相互转换

// 这个函数处理来自pool的消息，因为pool里面的消息不止来源于net层，也可能来源于其他逻辑层或者自身的延时发送消息
// 所以，这个函数里的消息不止有TCPTransferMsg，但是只需要处理TCPTransferMsg这个消息就行了
// 把从TCPTransferMsg解析出来的gate应该处理的报文返回，不处理的报文直接丢弃返回nil
func Gate_CreateCommonMsgByTCPTransferMsg(req proto.Message) proto.Message {
	switch data := req.(type) {
	case *protodf.TCPTransferMsg:
		fmt.Println("protodf.TCPTransferMsg base=", data.Base)
		switch data.DataKindId { //判断大类
		case uint32(protodf.CMDKindId_IDKindGate):
			switch data.DataSubId { //判断小类
			case uint32(protodf.CMDID_Gate_IDPulseReq):
				msg := new(protodf.PulseReq)
				err := proto.Unmarshal(data.Data, msg)
				protodf.SetBaseKindAndSubId(msg)
				protodf.CopyBaseExceptKindAndSubId(msg.Base, data.Base)
				if err != nil {
					fmt.Println("解析PulseReq出错")
					return nil
				}
				return msg
			case uint32(protodf.CMDID_Gate_IDPulseRsp):
				msg := new(protodf.PulseRsp)
				err := proto.Unmarshal(data.Data, msg)
				protodf.SetBaseKindAndSubId(msg)
				protodf.CopyBaseExceptKindAndSubId(msg.Base, data.Base)
				if err != nil {
					fmt.Println("解析PulseRsp出错")
					return nil
				}
				return msg
			case uint32(protodf.CMDID_Gate_IDTransferData):
				msg := new(protodf.GateTransferData)
				err := proto.Unmarshal(data.Data, msg)
				protodf.SetBaseKindAndSubId(msg)
				protodf.CopyBaseExceptKindAndSubId(msg.Base, data.Base)
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
		case uint32(protodf.CMDKindId_IDKindRouter):
			switch data.DataSubId { //判断小类
			case uint32(protodf.CMDID_Router_IDTransferDataRt):
				msg := new(protodf.RouterTransferData)
				err := proto.Unmarshal(data.Data, msg)
				protodf.SetBaseKindAndSubId(msg)
				protodf.CopyBaseExceptKindAndSubId(msg.Base, data.Base)
				if err != nil {
					fmt.Println("解析RouterTransferData出错")
					return nil
				}
				return msg
			default:
				fmt.Println("不识别的router报文，DataSubId=", data.DataSubId)
				return nil //丢弃这个TCPTransferMsg报文
			}
		default: //protodf.TCPTransferMsg报文的成员变量大类DataKindId不为CMDKindId_IDKindGate和CMDKindId_IDKindRouter都丢弃
			return nil //丢弃这个TCPTransferMsg报文
		}
	default: //*protodf.TCPTransferMsg以外的报文直接返回
		return req
	}
	return nil
}

// 这个函数处理发往pool的消息，因为除了普通消息以外还有kick掉一个session这样的消息
// 所以，这个函数里的要把kick消息区分出来，其他消息则转化成TCPTransferMsg
func Gate_CreateTCPTransferMsgByCommonMsg(req proto.Message, pBase *protodf.BaseInfo) proto.Message {
	switch data := req.(type) {
	case *protodf.TCPSessionKick:
		return data
	case *protodf.TCPTransferMsg:
		return data
	default:
		msg := new(protodf.TCPTransferMsg)
		protodf.SetBaseKindAndSubId(msg)
		protodf.CopyBaseExceptKindAndSubId(msg.Base, pBase)
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

func Gate_CreateCommonMsgByRouterTransferMsg(req *protodf.RouterTransferData) proto.Message {
	if req == nil {
		return nil
	}

	//大类
	switch req.DataCmdKind {
	case uint32(protodf.CMDKindId_IDKindClient):
		//小类
		switch req.DataCmdSubid {
		case uint32(protodf.CMDID_Client_IDLoginRsp):
			msg := new(protodf.LoginRsp)
			err := proto.Unmarshal(req.Data, msg)
			if err != nil {
				fmt.Println("LoginRsp解析失败")
				return nil
			}
			protodf.SetBaseKindAndSubId(msg)
			protodf.SetCommonMsgBaseByRouterTransferData(msg.Base, req)
			return msg
		default:
			return nil
		}
	default:
		return nil
	}

	return nil
}

func Gate_CreateGateTransferMsgByCommonMsg(req proto.Message) *protodf.GateTransferData {
	gateTrans := new(protodf.GateTransferData)
	protodf.SetBaseKindAndSubId(gateTrans)

	switch data := req.(type) {
	case *protodf.RouterTransferData:
		//RouterTransferData要特殊对待，不是调用proto.Marshal
		protodf.CopyBaseExceptKindAndSubId(gateTrans.Base, data.Base)
		gateTrans.Base.ConnId = data.AttGateconnid
		gateTrans.DataCmdKind = data.DataCmdKind
		gateTrans.DataCmdSubid = data.DataCmdSubid
		gateTrans.Data = make([]byte, len(data.Data))
		copy(gateTrans.Data, data.Data) //使用copy，让req可以被GateConnection
		gateTrans.AttAppid = data.SrcAppid
		gateTrans.AttApptype = data.SrcApptype
		gateTrans.ReqId = 0
	case *protodf.LoginRsp:
		fmt.Println("序列化protodf.LoginRsp, LoginRsp=", data)
		protodf.CopyBaseExceptKindAndSubId(gateTrans.Base, data.Base)
		buff, err := proto.Marshal(data)
		if err != nil {
			fmt.Println("LoginRsp序列化失败")
			return nil
		}
		gateTrans.Data = buff
		gateTrans.DataCmdKind = uint32(protodf.CMDKindId_IDKindClient)
		gateTrans.DataCmdSubid = uint32(protodf.CMDID_Client_IDLoginRsp)
		gateTrans.AttAppid = data.Base.AttAppid
		gateTrans.AttApptype = data.Base.AttApptype
		gateTrans.ClientRemoteAddress = data.Base.RemoteAdd
		fmt.Println("LoginRsp => gateTrans=", gateTrans)
	default:
		return nil
	}
	return gateTrans
}
