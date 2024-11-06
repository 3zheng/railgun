package main

import (
	"fmt"

	protodf "github.com/3zheng/railcommon/protodf"
	proto "google.golang.org/protobuf/proto"
)

//login消息过滤器,用于一般消息和protodf.RouterTransferData消息的相互转换

// 这个函数处理来自pool的消息，因为pool里面的消息不止来源于routeragent层，也可能来源于其他逻辑层或者自身的延时发送消息
// 所以，这个函数里的消息不止有RouterTransferData，但是只需要处理RouterTransferData这个消息就行了
// 把从RouterTransferData解析出来的login应该处理的报文返回，不处理的报文直接丢弃返回nil
func Login_CreateCommonMsgByRouterTransferData(req proto.Message) proto.Message {
	switch data := req.(type) {
	case *protodf.RouterTransferData:
		switch data.DataCmdKind { //判断大类
		case uint32(protodf.CMDKindId_IDKindClient):
			switch data.DataCmdSubid { //判断小类
			case uint32(protodf.CMDID_Client_IDLoginReq):
				msg := new(protodf.LoginReq)
				err := proto.Unmarshal(data.Data, msg)
				if err != nil {
					fmt.Println("解析LoginReq出错")
					return nil
				}
				protodf.SetBaseKindAndSubId(msg)
				protodf.CopyBaseExceptKindAndSubId(msg.Base, data.Base)
				msg.Base.AttAppid = data.SrcAppid
				msg.Base.AttApptype = data.SrcApptype
				msg.Base.RemoteAdd = data.ClientRemoteAddress
				msg.Base.GateConnId = data.AttGateconnid
				return msg
			default:
				fmt.Println("不识别的client报文，DataSubId=", data.DataCmdSubid)
				return nil //丢弃这个RouterTransferData报文
			}
		default: //protodf.TCPTransferMsg报文的成员变量大类DataKindId不为CMDKindId_IDKindGate都丢弃
			return nil //丢弃这个TCPTransferMsg报文
		}
	default: //*protodf.RouterTransferData以外的报文直接返回
		return req
	}
	return nil
}

// 这个函数处理发往pool的消息，因为除了普通消息以外还有kick掉一个session这样的消息
// 所以，这个函数里的要把kick消息区分出来，其他消息则转化成TCPTransferMsg
func Login_CreateRouterTransferDataByCommonMsg(req proto.Message, pBase *protodf.BaseInfo) proto.Message {
	switch data := req.(type) {
	case *protodf.TCPTransferMsg: //实际上不应该传TCPTransferMsg类型的报文到这个函数里来
		return data
	case *protodf.RouterTransferData: //实际上不应该传RouterTransferData类型的报文到这个函数里来
		return data
	default:
		msg := new(protodf.RouterTransferData)
		protodf.SetBaseKindAndSubId(msg)
		protodf.CopyBaseExceptKindAndSubId(msg.Base, pBase)
		msg.DataCmdKind = pBase.KindId
		msg.DataCmdSubid = pBase.SubId
		msg.DestAppid = pBase.AttAppid
		msg.DestApptype = pBase.AttApptype
		msg.AttGateconnid = pBase.GateConnId
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
