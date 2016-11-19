// DialManager project DialManager.go
package DialManager

//导入TcpListenManager是为了借用粘包处理的函数
import (
	serverManger "TcpListenManager"
	"bytes"
	"fmt"
	"net"
	bs_tcp "protodefine/tcpnet"

	proto "github.com/golang/protobuf/proto"
)

//作为客户端去连接
type ConnectionSession struct {
	//	quitWriteCh  chan int                   //通知结束客户端write协程的管道
	remoteAddr string                      //对端地址包括ip和port
	connId     uint64                      //连接标识id
	tcpConn    *net.TCPConn                //用于发送接收消息的tcp连接实体
	cache      *bytes.Buffer               //自动扩展的用于粘包处理的缓存区
	MsgWriteCh chan *bs_tcp.TCPTransferMsg //从接受来自逻辑层消息的管道
	Quit       chan bool
}

func (session *ConnectionSession) RecvPackege(logicChannel chan proto.Message) {
	data := make([]byte, 1024) //1024字节为一个数据片
	//(*session).cache.
	//不停的读取对端remote服务器发来的信息
	for {
		dataLength, err := (*session).tcpConn.Read(data)
		if err != nil {
			fmt.Println("TCP读取数据错误:", err.Error())
			//向MsgWriteCh随意发送一个报文让来结束当前的来SendPackege的MsgWriteCh管道阻塞从而让其conn.send失败而关闭SendPackege协程
			kick := new(bs_tcp.TCPTransferMsg)
			session.tcpConn.Close()    //关闭TCPCONN
			session.MsgWriteCh <- kick //让SendPackege结束MsgWriteCh的阻塞
			session.Quit <- true       //通知主线程连接断开了
			break                      //跳出for循环
		} else {
			buff := data[:dataLength] //指示有效数据长度
			validBuff := serverManger.DecodePackage(session.cache, buff)
			if validBuff == nil {
				continue //报文体没有收完整，还需要继续read
			}
			protoMsg := new(bs_tcp.TCPTransferMsg)
			err := proto.Unmarshal(validBuff, protoMsg)
			if err != nil {
				fmt.Println("反序列化TCPTransferMsg报文出错，无法解析validBuff, err=", err.Error())
				//可能消息出错了清空cache
				session.cache.Reset()
			} else {
				//发送报文给逻辑层
				protoMsg.Base.ConnId = session.connId
				logicChannel <- protoMsg
			}
		}
	}

}

//向客户端发消息
func (session *ConnectionSession) SendPackege(logicChannel chan proto.Message) {
	//等待从逻辑层下发的消息
	quit := false
	for {
		if quit {
			break //在select case里break是跳不出循环的
		}
		select {
		case v, ok := <-session.MsgWriteCh:
			if ok {
				data, err := proto.Marshal(v)
				if err != nil {
					fmt.Println("序列化TCPTransferMsg报文出错")
				} else {
					var pkg *bytes.Buffer = new(bytes.Buffer)
					serverManger.EncodePackage(pkg, data) //调用DealStickPkg.go的函数，用于组一个（报文长度+报文）的包
					_, err2 := session.tcpConn.Write(pkg.Bytes())
					if err2 != nil {
						//直接tcpConn.close就行了，因为RecvPackege协程是阻塞在read处的，这里close了，read就能得到error异常了
						fmt.Println("TCP写数据错误:", err2.Error())
						session.tcpConn.Close()
						session.Quit <- true //通知主线程连接断开了
						quit = true
					}
				}
			} else {
				//如果关闭了这个管道，说明需要session需要关闭
				quit = true
				fmt.Println("connId=", session.connId, "的会话已经退出send协程")
			}
		}
	}
}

func CreateClient(remote_address string, logicChannel chan proto.Message) *ConnectionSession {
	conn, err := net.Dial("tcp", remote_address)
	if err != nil {
		fmt.Println("连接服务端失败:", err.Error())
		return nil
	}
	var tcpConn *net.TCPConn
	switch myconn := conn.(type) {
	case *net.TCPConn:
		fmt.Println("已建立起连接的服务器TCP连接，对端地址=", myconn.RemoteAddr().String())
		tcpConn = myconn
	default:
		return nil
	}

	fmt.Println("DialManager connected : ", tcpConn.RemoteAddr().String())
	session := new(ConnectionSession)
	session.remoteAddr = tcpConn.RemoteAddr().String()
	session.tcpConn = tcpConn
	session.cache = new(bytes.Buffer)
	session.MsgWriteCh = make(chan *bs_tcp.TCPTransferMsg, 10000) //可以存放10000个报文
	session.Quit = make(chan bool)
	//不需要传递TCPSessionCome给逻辑层，因为这个会话是不需要管理的
	go session.RecvPackege(logicChannel)
	go session.SendPackege(logicChannel)

	return session
}
