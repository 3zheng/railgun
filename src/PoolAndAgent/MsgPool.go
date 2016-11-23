//PoolAndAgent project MsgPool.go
package PoolAndAgent

import (
	"fmt"
	ListenManager "railgun/TcpListenManager"
	bs_proto "railgun/protodefine"
	bs_types "railgun/protodefine/mytype"
	bs_router "railgun/protodefine/router"
	bs_tcp "railgun/protodefine/tcpnet"
	"sync/atomic"
	"time"

	proto "github.com/golang/protobuf/proto"
)

const CHANNEL_LENGTH = 100000 //消息管道的容量
const ONPULSE_INTERVAL = 100  //LogicProcess的OnPulse函数定时调用的时间间隔，单位毫秒

type SingleMsgPool struct {
	quit   chan int //结束程序运行报文,传输的是appid
	IsInit bool
	//NetToLogicChannel如果不和服务端监听的网络层绑定，这bindingNetAgent和NetToLogicChannel两个成员变量就为空
	NetToLogicChannel chan proto.Message //网络层写，Pool层读后往逻辑层发送的消息管道
	bindingNetAgent   *NetAgent          //与pool绑定的NetAgent网络层
	//与router相连的网络层，bindingRouterAgent除了router app本身没有，其他app都会有，因为所有app若与其他app通信都是要靠router来转发
	RouterToLogicChannel chan proto.Message
	bindingRouterAgent   *RouterAgent
	//数据库层,但是不像netAgent层和RouterAgent层是有独立的channel绑定，所以不叫agent。应该相当于ILogicProcess的附属层，绑定在ILogicProcess上的
	//如果不连数据库的话没必要有
	bindingDBProcess [](*CADODatabase) //如果有的话和bindingLogicProcesses一一对应
	//ILogicProcess是一定有的，pool必须绑定，不然从网络层收来的报文就没法做业务处理了
	PoolToLogicChannel    chan proto.Message //Pool层写逻辑层读的消息管道
	bindingLogicProcesses []ILogicProcess    //与pool绑定的逻辑处理层
	myAppType             uint32
	myAppId               uint32
}

//创建一个MsgPool
func CreateMsgPool(quit chan int, myAppType uint32, myAppId uint32) *SingleMsgPool {
	pool := new(SingleMsgPool)
	pool.IsInit = false
	pool.bindingNetAgent = nil
	pool.quit = quit
	pool.myAppType = myAppType
	pool.myAppId = myAppId
	bs_proto.OutputMyLog("CreateMsgPool() myAppType=", myAppType)
	//fmt.Println("CreateMsgPool() myAppType=", myAppType)
	return pool
}

//结束程序运行
func (this *SingleMsgPool) StopRun() {
	if this.quit != nil {
		this.quit <- 1
	}
}

//this并不是golang关键字
/*初始化并运行
//初始化并运行，如果bindingLogicProcesses的len大于1的话，而且需要对每个ILogicProcess都进行额外的初始化，那么InitMsg这个就是初始化报文
//比如数据库处理的pool就需要这么初始化，因为数据库的IO比较慢，所以需要多协程。
//假设我开了10个数据库协程，如果我不一开始把InitMsg推给每个ILogicProcess都处理的话，后面ILogicProcess就无法全部都初始化了
//因为10个ILogicProcess协程是共同读取同一个PoolToLogicChannel管道，所以如果像主逻辑协程那样在初始化后再发送初始化报文，那么如果连续发送10个报文
//可能是有些协程重复收到了初始化报文，有些协程没有收到初始化报文，所以一定要在go RunLogicProcess把每个协程ILogicProcess都初始化
*/
func (this *SingleMsgPool) InitAndRun(InitMsg proto.Message) bool {
	if len(this.bindingLogicProcesses) == 0 {
		fmt.Println("pool必须绑定逻辑处理实例")
		return false
	}
	//先创建PoolToLogicChannel管道
	this.PoolToLogicChannel = make(chan proto.Message, CHANNEL_LENGTH)
	//判断有没有绑定netagent，如果有那么要创建相应的tcpManager和NetToLogicChannel
	if this.bindingNetAgent != nil {
		this.NetToLogicChannel = make(chan proto.Message, CHANNEL_LENGTH)
		go ListenManager.CreateSever(this.bindingNetAgent.IpAddress, this.NetToLogicChannel) //创建服务端监听协程
	}
	if this.bindingRouterAgent != nil {
		this.RouterToLogicChannel = make(chan proto.Message, CHANNEL_LENGTH)
		go this.bindingRouterAgent.RunRouterAgent(this.RouterToLogicChannel, this.myAppId, this.myAppType) //调用这个有断线重连机制的函数
	}
	//一个协程不停的读取NetToLogicChannel然后写入PoolToLogicChannel
	if this.NetToLogicChannel != nil {
		go func() {
			for {
				select {
				case v := <-this.NetToLogicChannel: //阻塞读取NetToLogicChannel,来自NetToLogicChannel的报文都在属于tcp.proto
					switch data := v.(type) {
					case *bs_tcp.TCPSessionKick: //来自NetToLogicChannel的kick报文要做特殊处理，不上传给logic层
						sess := ListenManager.GetSessionByConnId(data.Base.ConnId)
						if sess != nil && atomic.LoadInt32(&(sess.IsSendKickMsg)) == 0 { //原子操作，判断sess.IsSendKickMsg == 0
							go sess.CloseSession(this.NetToLogicChannel) //关闭此session
						}
					case *bs_tcp.TCPSessionCome:
						fmt.Println("收到了TCPSessionCome报文，base=", data.Base)
						fmt.Println("NetToLogicChannel转发给PoolToLogicChannel")
						select {
						case this.PoolToLogicChannel <- v:
						case <-time.After(5 * time.Second):
							fmt.Println("严重事件，PoolToLogicChannel已经阻塞超时5秒")
						}
					default: //其他报文都直接丢给PoolToLogicChannel管道
						fmt.Println("NetToLogicChannel转发给PoolToLogicChannel")
						select {
						case this.PoolToLogicChannel <- v:
						case <-time.After(5 * time.Second):
							fmt.Println("严重事件，PoolToLogicChannel已经阻塞超时5秒")
						}
					}
				}
			}
		}()
	}
	if this.RouterToLogicChannel != nil {
		//一个协程不停的读取RouterToLogicChannel然后写入PoolToLogicChannel
		go func() {
			for {
				select {
				case v := <-this.RouterToLogicChannel: //阻塞读取RouterToLogicChannel,来自RouterToLogicChannel的报文都在router.proto
					switch data := v.(type) {
					case *bs_tcp.TCPSessionKick: //来自routerAgent的报文kick,come,close报文都是无意义的
						fmt.Println("收到了无意义的TCPSessionKick报文")
					case *bs_tcp.TCPSessionCome:
						fmt.Println("收到了无意义的TCPSessionCome报文")
					case *bs_tcp.TCPSessionClose:
						fmt.Println("收到了无意义的TCPSessionCome报文")
					case *bs_tcp.TCPTransferMsg:
						if data.DataKindId != uint32(bs_types.CMDKindId_IDKindRouter) {
							break //跳出当前的case
						}
						//先把TCPTransferMsg转成RouterTransferData再传给PoolToLogicChannel
						//因为从router app来的有用报文只有RouterTransferData
						var pRouterTran *bs_router.RouterTransferData = nil
						switch data.DataSubId {
						case uint32(bs_router.CMDID_Router_IDRegisterAppRsp):
							msg := new(bs_router.RegisterAppRsp)
							err := proto.Unmarshal(data.Data, msg)
							if err != nil {
								break
							}
							fmt.Println("收到注册回复，RegResult=", msg.RegResult)
						case uint32(bs_router.CMDID_Router_IDTransferData):
							pRouterTran = new(bs_router.RouterTransferData)
							err := proto.Unmarshal(data.Data, pRouterTran)
							if err != nil {
								break
							}
						}
						if pRouterTran == nil {
							break
						}
						//写RouterTransferData到PoolToLogicChannel中
						fmt.Println("RouterToLogicChannel转发给PoolToLogicChannel")
						select {
						case this.PoolToLogicChannel <- pRouterTran:
						case <-time.After(5 * time.Second):
							fmt.Println("严重事件，PoolToLogicChannel已经阻塞超时5秒")
						}
					default: //其他报文都直接丢给PoolToLogicChannel管道
						fmt.Println("收到了未知报文")
					}
				}
			}
		}()
	}
	//根据需要创建的协程数，创建不停的读取PoolToLogicChannel的协程，在执行了init函数后，循环执行ProcessReq()函数和OnPluse()定时函数
	for i, logic := range this.bindingLogicProcesses {
		var pDB *CADODatabase = nil
		if len(this.bindingDBProcess) != 0 && len(this.bindingDBProcess) == len(this.bindingLogicProcesses) {
			//不为0的话，数量必须相等，不然就出错了
			pDB = this.bindingDBProcess[i]
		}
		go this.RunLogicProcess(logic, pDB, InitMsg)
	}
	this.IsInit = true
	return true
}

func (this *SingleMsgPool) RunLogicProcess(pLogic ILogicProcess, pDataBase *CADODatabase, InitMsg proto.Message) {
	pLogic.Init(this) //初始化
	//在创建协程时，让每个协程的logic先预处理InitMsg初始化报文
	if InitMsg != nil {
		pLogic.ProcessReq(InitMsg, nil)
	}
	//阻塞读取PoolToLogicChannel在取得报文后调用ProcessReq和定时调用OnPulse
	t1 := time.NewTimer(ONPULSE_INTERVAL * time.Millisecond)
	var nMs uint64 = uint64(ONPULSE_INTERVAL)
	for {
		select {
		case v, ok := <-this.PoolToLogicChannel:
			if ok {
				pLogic.ProcessReq(v, pDataBase)
			}
		case <-t1.C:
			pLogic.OnPulse(nMs)
			t1.Reset(ONPULSE_INTERVAL * time.Millisecond)
			nMs += uint64(ONPULSE_INTERVAL)
		}
	}
}

func (this *SingleMsgPool) BindNetAgent(agent *NetAgent) {
	this.bindingNetAgent = agent
}

func (this *SingleMsgPool) BindRouterAgent(agent *RouterAgent) {
	this.bindingRouterAgent = agent
}

func (this *SingleMsgPool) AddDataBaseProcess(process *CADODatabase) {
	this.bindingDBProcess = append(this.bindingDBProcess, process)
	process.InitDB() //这个InitDB并没有去连接数据库
}

func (this *SingleMsgPool) AddLogicProcess(agent ILogicProcess) {
	this.bindingLogicProcesses = append(this.bindingLogicProcesses, agent)
}

//延时nMs毫秒后推送到PoolToLogicChannel队列中
func (this *SingleMsgPool) PushMsg(req proto.Message, nMs uint64) {
	//因为在实际业务中需要延时发送的报文不多所以如果有延时发送另起一个协程
	if nMs != 0 {
		go func(delayTime uint64) {
			select {
			//阻塞delayTime * time.Millisecond后向PoolToLogicChannel发送req
			case <-time.After(time.Duration(delayTime) * time.Millisecond):
				fmt.Println("业务逻辑延时发给PoolToLogicChannel")
				this.PoolToLogicChannel <- req //因为是起了一个协程来发送，所以这里就算阻塞了也没事，就不判断超时事件了
			}
		}(nMs)
	} else { //立即推送进channel
		fmt.Println("业务逻辑直接立即发给PoolToLogicChannel")
		select {
		case this.PoolToLogicChannel <- req:
		case <-time.After(5 * time.Second):
			fmt.Println("严重事件，PoolToLogicChannel已经阻塞超时5秒")
		}
	}
}

func (this *SingleMsgPool) SendMsgToClientByNetAgent(req proto.Message) {
	if this.bindingNetAgent != nil {
		switch data := req.(type) {
		case *bs_tcp.TCPTransferMsg:
			this.bindingNetAgent.SendMsg(data)
		case *bs_tcp.TCPSessionKick:
			sess := ListenManager.GetSessionByConnId(data.Base.ConnId)
			if sess != nil && atomic.LoadInt32(&(sess.IsSendKickMsg)) == 0 { //原子操作
				go sess.CloseSession(this.NetToLogicChannel) //关闭此session
			}
		default:
			fmt.Println("不处理TCPSessionKick和TCPTransferMsg以外的报文")
		}
	}
}

func (this *SingleMsgPool) SendMsgToServerAppByRouter(req proto.Message) {
	if this.bindingRouterAgent != nil {
		switch data := req.(type) {
		case *bs_tcp.TCPTransferMsg:
			this.bindingRouterAgent.SendMsg(data)
		case *bs_router.RouterTransferData:
			//先把RouterTransferData转成TCPTransferMsg再发送
			buff, err := proto.Marshal(data)
			if err != nil {
				break
			}
			tcpTran := new(bs_tcp.TCPTransferMsg)
			bs_proto.SetBaseKindAndSubId(tcpTran)
			bs_proto.CopyBaseExceptKindAndSubId(tcpTran.Base, data.Base)
			tcpTran.Data = buff
			tcpTran.DataKindId = uint32(bs_types.CMDKindId_IDKindRouter)
			tcpTran.DataSubId = uint32(bs_router.CMDID_Router_IDTransferData)
			this.bindingRouterAgent.SendMsg(tcpTran)
		default:
			fmt.Println("不处理TCPTransferMsg和RouterTransferData以外的报文")
		}
	}
}
