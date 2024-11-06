// ClientForTest project main.go
package main

import (
	"fmt"

	"github.com/3zheng/railcommon"
	protodf "github.com/3zheng/railproto"
	proto "google.golang.org/protobuf/proto"
)

func main() {
	//创建连接GATE的客户端
	ch := make(chan proto.Message, 1000)
	session := railcommon.CreateClient("127.0.0.1:4101", ch)
	if session == nil {
		fmt.Println("连接gate失败")
		return
	}
	go ReceiveMsg(ch)
	//发送登录请求
	loginReq := new(protodf.LoginReq)
	loginReq.LoginAccount = "yourname"
	loginReq.LoginPassword = "E10ADC3949BA59ABBE56E057F20F883E" //123456的MD5加密
	tcpMsg := ChangeCommonMsgToTCPTransferMsg(loginReq)
	if tcpMsg != nil {
		session.MsgWriteCh <- tcpMsg //向gate发送消息
	}
	fmt.Println("按任意键结束程序")
	sms := make([]byte, 50)
	_, err := fmt.Scan(&sms)
	if err != nil {
		fmt.Println("scan出错,err=", err)
	}
}

func ChangeCommonMsgToTCPTransferMsg(common proto.Message) *protodf.TCPTransferMsg {
	tcpMsg := new(protodf.TCPTransferMsg)
	gateMsg := new(protodf.GateTransferData)
	switch data := common.(type) {
	case *protodf.LoginReq:
		buff, err := proto.Marshal(data)
		if err != nil {
			return nil
		}
		gateMsg.Data = buff
		gateMsg.DataCmdKind = uint32(protodf.CMDKindId_IDKindClient)
		gateMsg.DataCmdSubid = uint32(protodf.CMDID_Client_IDLoginReq)
		gateMsg.AttApptype = uint32(protodf.EnumAppType_Login)   //目标server app为登录app
		gateMsg.AttAppid = uint32(protodf.EnumAppId_Send2AnyOne) //随意一个登录app就行了，不需要知道其APPID
	default:
		fmt.Println("不识别的报文")
		return nil
	}
	tcpBuff, err := proto.Marshal(gateMsg)
	if err != nil {
		return nil
	}
	tcpMsg.Data = tcpBuff
	tcpMsg.DataKindId = uint32(protodf.CMDKindId_IDKindGate)
	tcpMsg.DataSubId = uint32(protodf.CMDID_Gate_IDTransferData)
	return tcpMsg
}

// 循环阻塞读取
func ReceiveMsg(ch chan proto.Message) {
	for {
		select {
		case v, ok := <-ch:
			if !ok {
				return //跳出函数
			}
			switch data := v.(type) {
			case *protodf.TCPTransferMsg:
				fmt.Println("收到了gate的回复报文")
				//处理protodf.GateTransferData
				if data.DataKindId == uint32(protodf.CMDKindId_IDKindGate) && data.DataSubId == uint32(protodf.CMDID_Gate_IDTransferData) {
					fmt.Println("收到了protodf.GateTransferData")
					gateMsg := new(protodf.GateTransferData)
					err := proto.Unmarshal(data.Data, gateMsg)
					if err != nil {
						fmt.Println("proto反序列化失败")
						break
					}
					fmt.Println("attAppType=", gateMsg.AttApptype, "attAppID=", gateMsg.AttAppid)
					fmt.Println("gateMsg", gateMsg)
					if gateMsg.DataCmdKind == uint32(protodf.CMDKindId_IDKindClient) && gateMsg.DataCmdSubid == uint32(protodf.CMDID_Client_IDLoginRsp) {
						//登录回复
						fmt.Println("收到了protodf.LoginRsp报文")
						loginRsp := new(protodf.LoginRsp)
						err := proto.Unmarshal(gateMsg.Data, loginRsp)
						if err != nil {
							fmt.Println("proto反序列化失败")
							break
						}
						fmt.Println("loginRsp=", loginRsp)
						fmt.Println("收到登录回复，LoginResult=", loginRsp.LoginResult, ",UserId=", loginRsp.UserId)
					}
				}
			default:
				fmt.Println("收到了protodf.TCPTransferMsg以外的报文")
			}
		}
	}
}
