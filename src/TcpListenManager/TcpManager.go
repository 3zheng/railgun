// TcpManager project TcpManager.go
package TcpManager

import (
	"bytes"
	"fmt"
	"net"
	bs_proto "protodefine"
	bs_tcp "protodefine/tcpnet"
	"sync"
	"sync/atomic"

	proto "github.com/golang/protobuf/proto"
)

var gSessionLock sync.Mutex //gSessionMap的临界区锁
var gSessionMap map[uint64]*ConnectionSession

//服务端的
type ConnectionSession struct {
	//	quitWriteCh  chan int                   //通知结束客户端write协程的管道
	remoteAddr    string                      //对端地址包括ip和port
	connId        uint64                      //连接标识id
	tcpConn       *net.TCPConn                //用于发送接收消息的tcp连接实体
	cache         *bytes.Buffer               //自动扩展的用于粘包处理的缓存区
	MsgWriteCh    chan *bs_tcp.TCPTransferMsg //从接受来自逻辑层消息的管道
	wg            sync.WaitGroup              //在close
	isWriteClosed int32                       //SendPackege协程是否已经结束的标志位
	isReadClosed  int32                       //RecvPackege协程是否已经结束的标志位
	IsSendKickMsg int32                       //MsgPool是否处理了Kick报文或者说是否已经调用了CloseSession的标志位,MsgPool也会访问，所以大写
}

//应对粘包采用的数据格式是4个字节的int32类型的length变量作为包头，后续跟上长度为length的包实体
//从客户端收消息
func (session *ConnectionSession) RecvPackege(logicChannel chan proto.Message) {
	data := make([]byte, 1024) //1024字节为一个数据片
	//(*session).cache.
	//不停的读取客户端发来的信息
	for {
		dataLength, err := (*session).tcpConn.Read(data)
		if err != nil {
			fmt.Println("读取客户端数据错误:", err.Error())
			//向主线程发送TCPSessionKick报文让主线程来关闭SendPackege协程
			kick := new(bs_tcp.TCPSessionKick)
			bs_proto.SetBaseKindAndSubId(kick) //目前只set了kind_id和sub_id
			kick.Base.ConnId = session.connId
			logicChannel <- kick //通知主线程发送断开当前session
			break                //跳出for循环
		} else {
			buff := data[:dataLength] //指示有效数据长度
			validBuff := DecodePackage(session.cache, buff)
			if validBuff == nil {
				continue //报文体没有收完整，还需要继续read
			}
			protoMsg := new(bs_tcp.TCPTransferMsg)
			err := proto.Unmarshal(validBuff, protoMsg)
			bs_proto.SetBaseKindAndSubId(protoMsg) //目前只set了kind_id和sub_id，从客户端发来的base可能为nil,Unmarshal后要重新赋值
			if err != nil {
				fmt.Println("反序列化TCPTransferMsg报文出错，无法解析validBuff, err=", err.Error())
				//可能消息出错了清空cache
				session.cache.Reset()
			} else {
				//发送报文给逻辑层
				protoMsg.Base.ConnId = session.connId
				protoMsg.Base.GateConnId = session.connId
				protoMsg.Base.RemoteAdd = session.remoteAddr
				logicChannel <- protoMsg
			}
		}
	}

	atomic.StoreInt32(&(session.isReadClosed), 1) //协程间公用标志位使用原子操作
	session.wg.Done()
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
					EncodePackage(pkg, data) //调用DealStickPkg.go的函数，用于组一个（报文长度+报文）的包
					_, err2 := session.tcpConn.Write(pkg.Bytes())
					if err2 != nil {
						//向主线程发送TCPSessionKick报文让主线程来关闭RecvPackege协程
						kick := new(bs_tcp.TCPSessionKick)
						bs_proto.SetBaseKindAndSubId(kick) //目前只set了kind_id和sub_id
						kick.Base.ConnId = session.connId
						logicChannel <- kick //通知主线程发送断开当前session
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
	atomic.StoreInt32(&(session.isWriteClosed), 1) //协程间公用标志位使用原子操作
	session.wg.Done()
}

//CloseSession必须由逻辑层创建协程来调用，因为这里使用了session.wg.Wait()阻塞,而且必须先判断之前是否已经调用了CloseSession，不能调用两次
func (session *ConnectionSession) CloseSession(logicChannel chan proto.Message) {
	ret := atomic.CompareAndSwapInt32(&(session.IsSendKickMsg), 0, 1)
	if !ret {
		fmt.Println("已经CloseSession被调用了两次，connId=", session.connId)
	}
	if v := atomic.LoadInt32(&(session.isWriteClosed)); v == 0 {
		close(session.MsgWriteCh) //关闭SendPackege接受消息的管道，以结束SendPackege协程
	}

	if v := atomic.LoadInt32(&(session.isReadClosed)); v == 0 {
		session.tcpConn.Close() //关闭自身的net.TCPConn，让RecvPackege结束tcpConn.Read阻塞，以结束RecvPackege协程
	}

	session.wg.Wait() //阻塞CloseSession直到读写两个协程都结束
	connId := session.connId
	//然后删除map中对应的元素
	gSessionLock.Lock()
	delete(gSessionMap, session.connId)
	gSessionLock.Unlock()
	//向逻辑层发送session关闭的通知
	msg := new(bs_tcp.TCPSessionClose)
	bs_proto.SetBaseKindAndSubId(msg)
	msg.Base.ConnId = connId
	logicChannel <- msg
	fmt.Println("gSessionMap已删除connId=", connId, "的session")
}

//ip_address是带端口的字符串 比如 127.0.0.1:8008, logicChannel是将网络层消息传递给逻辑层主线程的管道
func CreateSever(ip_address string, logicChannel chan proto.Message) {
	//listen, err := net.ListenTCP("tcp", &net.TCPAddr{net.ParseIP(ip), port, ""})
	var tcpAddr *net.TCPAddr
	tcpAddr, _ = net.ResolveTCPAddr("tcp", ip_address)
	listener, err := net.ListenTCP("tcp", tcpAddr)
	//listen, err := net.Listen("tcp", ip_address) //如果使用这种TCP监听，返回值是type net.Listener类型需要强制转net.TCPListener类型才可以进行AcceptTCP操作
	if err != nil {
		fmt.Println("监听端口失败:", err.Error())
		return
	}
	fmt.Println("已初始化连接，等待客户端连接...")
	//defer listener.Close()//一直在循环就不要关闭listener了
	var currentConnId uint64 = 0 //connId计数，当accept了一个客户端，就+1
	gSessionMap = make(map[uint64]*ConnectionSession)
	//无限循环accept
	for {
		tcpConn, err := listener.AcceptTCP()
		if err != nil {
			fmt.Println("accepct错误:", err.Error())
			continue
		}
		currentConnId += 1
		fmt.Println("A client connected : ", tcpConn.RemoteAddr().String(), "currentConnId=", currentConnId)
		session := new(ConnectionSession)
		session.remoteAddr = tcpConn.RemoteAddr().String()
		session.connId = currentConnId
		session.tcpConn = tcpConn
		session.cache = new(bytes.Buffer)
		session.MsgWriteCh = make(chan *bs_tcp.TCPTransferMsg, 100) //可以存放100个报文
		session.isReadClosed = 0
		session.isWriteClosed = 0
		session.IsSendKickMsg = 0
		gSessionLock.Lock()
		gSessionMap[currentConnId] = session
		gSessionLock.Unlock()
		//创建读写函数goroutine协程，logicChannel有两个作用，第一传递由网络层自己产生的kindId为NetWork的消息给主线程，第二传递客户端发过来的消息给主线程
		session.wg.Add(2)
		go session.RecvPackege(logicChannel) //RecvPackege的logicChannel作用1、2都有
		go session.SendPackege(logicChannel) //SendPackege的logicChannel只有作用1
		msg := new(bs_tcp.TCPSessionCome)
		bs_proto.SetBaseKindAndSubId(msg)
		msg.Base.ConnId = session.connId
		msg.Base.RemoteAdd = session.remoteAddr
		logicChannel <- msg
		fmt.Println("向logicChannel发送了TCPSessionCome报文")
	}
}

func GetSessionByConnId(connId uint64) *ConnectionSession {
	if elem, ok := gSessionMap[connId]; ok == true {
		return elem
	} else {
		return nil
	}
}
