package main

import (
	"fmt"

	protodf "github.com/3zheng/railcommon/protodf"
	proto "google.golang.org/protobuf/proto"
)

//gate消息过滤器,用于一般消息和protodf.TCPTransferMsg消息的相互转换

// 这个函数处理来自pool的消息，因为pool里面的消息不止来源于net层，也可能来源于其他逻辑层或者自身的延时发送消息
// 所以，这个函数里的消息不止有TCPTransferMsg，但是只需要处理TCPTransferMsg这个消息就行了
// 把从TCPTransferMsg解析出来的gate应该处理的报文返回，不处理的报文直接丢弃返回nil
func Router_CreateCommonMsgByTCPTransferMsg(req proto.Message) proto.Message {
	switch data := req.(type) {
	case *protodf.TCPTransferMsg:
		switch data.DataKindId { //判断大类
		case uint32(protodf.CMDKindId_IDKindRouter):
			switch data.DataSubId { //判断小类
			case uint32(protodf.CMDID_Router_IDTransferData):
				msg := new(protodf.RouterTransferData)
				err := proto.Unmarshal(data.Data, msg)
				protodf.SetBaseKindAndSubId(msg)
				protodf.CopyBaseExceptKindAndSubId(msg.Base, data.Base)
				if err != nil {
					fmt.Println("解析PulseReq出错")
					return nil
				}
				return msg
			case uint32(protodf.CMDID_Router_IDRegisterAppReq):
				msg := new(protodf.RegisterAppReq)
				err := proto.Unmarshal(data.Data, msg)
				protodf.SetBaseKindAndSubId(msg)
				protodf.CopyBaseExceptKindAndSubId(msg.Base, data.Base)
				if err != nil {
					fmt.Println("解析PulseReq出错")
					return nil
				}
				return msg
			case uint32(protodf.CMDID_Router_IDRegisterAppRsp):
				msg := new(protodf.RegisterAppRsp)
				err := proto.Unmarshal(data.Data, msg)
				protodf.SetBaseKindAndSubId(msg)
				protodf.CopyBaseExceptKindAndSubId(msg.Base, data.Base)
				if err != nil {
					fmt.Println("解析PulseReq出错")
					return nil
				}
				return msg
			default:
				fmt.Println("不识别的gate报文，DataSubId=", data.DataSubId)
				return nil //丢弃这个TCPTransferMsg报文
			}
		default: //protodf.TCPTransferMsg报文的成员变量大类DataKindId不为CMDKindId_IDKindGate都丢弃
			return nil //丢弃这个TCPTransferMsg报文
		}
	default: //*protodf.TCPTransferMsg以外的报文直接返回
		return req
	}
	return nil
}

// 这个函数处理发往pool的消息，因为除了普通消息以外还有kick掉一个session这样的消息
// 所以，这个函数里的要把kick消息区分出来，其他消息则转化成TCPTransferMsg
func Router_CreateTCPTransferMsgByCommonMsg(req proto.Message, pBase *protodf.BaseInfo) proto.Message {
	switch data := req.(type) {
	case *protodf.TCPSessionKick:
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
