// ClientForTest project main.go
package main

import (
	"fmt"

	"github.com/3zheng/railgun/DialManager"
	bs_client "github.com/3zheng/railgun/protodefine/client"
	bs_gate "github.com/3zheng/railgun/protodefine/gate"
	bs_types "github.com/3zheng/railgun/protodefine/mytype"
	bs_tcp "github.com/3zheng/railgun/protodefine/tcpnet"
	proto "github.com/golang/protobuf/proto"
)

func main() {
	//创建连接GATE的客户端
	ch := make(chan proto.Message, 1000)
	session := DialManager.CreateClient("127.0.0.1:4101", ch)
	if session == nil {
		fmt.Println("连接gate失败")
		return
	}
	go ReceiveMsg(ch)
	//发送登录请求
	loginReq := new(bs_client.LoginReq)
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

func ChangeCommonMsgToTCPTransferMsg(common proto.Message) *bs_tcp.TCPTransferMsg {
	tcpMsg := new(bs_tcp.TCPTransferMsg)
	gateMsg := new(bs_gate.TransferData)
	switch data := common.(type) {
	case *bs_client.LoginReq:
		buff, err := proto.Marshal(data)
		if err != nil {
			return nil
		}
		gateMsg.Data = buff
		gateMsg.DataCmdKind = uint32(bs_types.CMDKindId_IDKindClient)
		gateMsg.DataCmdSubid = uint32(bs_client.CMDID_Client_IDLoginReq)
		gateMsg.AttApptype = uint32(bs_types.EnumAppType_Login)   //目标server app为登录app
		gateMsg.AttAppid = uint32(bs_types.EnumAppId_Send2AnyOne) //随意一个登录app就行了，不需要知道其APPID
	default:
		fmt.Println("不识别的报文")
		return nil
	}
	tcpBuff, err := proto.Marshal(gateMsg)
	if err != nil {
		return nil
	}
	tcpMsg.Data = tcpBuff
	tcpMsg.DataKindId = uint32(bs_types.CMDKindId_IDKindGate)
	tcpMsg.DataSubId = uint32(bs_gate.CMDID_Gate_IDTransferData)
	return tcpMsg
}

//循环阻塞读取
func ReceiveMsg(ch chan proto.Message) {
	for {
		select {
		case v, ok := <-ch:
			if !ok {
				return //跳出函数
			}
			switch data := v.(type) {
			case *bs_tcp.TCPTransferMsg:
				fmt.Println("收到了gate的回复报文")
				//处理bs_gate.TransferData报文
				if data.DataKindId == uint32(bs_types.CMDKindId_IDKindGate) && data.DataSubId == uint32(bs_gate.CMDID_Gate_IDTransferData) {
					fmt.Println("收到了bs_gate.TransferData报文")
					gateMsg := new(bs_gate.TransferData)
					err := proto.Unmarshal(data.Data, gateMsg)
					if err != nil {
						fmt.Println("proto反序列化失败")
						break
					}
					fmt.Println("attAppType=", gateMsg.AttApptype, "attAppID=", gateMsg.AttAppid)
					fmt.Println("gateMsg", gateMsg)
					if gateMsg.DataCmdKind == uint32(bs_types.CMDKindId_IDKindClient) && gateMsg.DataCmdSubid == uint32(bs_client.CMDID_Client_IDLoginRsp) {
						//登录回复
						fmt.Println("收到了bs_client.LoginRsp报文")
						loginRsp := new(bs_client.LoginRsp)
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
				fmt.Println("收到了bs_tcp.TCPTransferMsg以外的报文")
			}
		}
	}
}
