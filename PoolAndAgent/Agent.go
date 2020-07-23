package PoolAndAgent

import (
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/3zheng/railgun/DialManager"
	ListenManager "github.com/3zheng/railgun/TcpListenManager"
	bs_proto "github.com/3zheng/railgun/protodefine"
	bs_types "github.com/3zheng/railgun/protodefine/mytype"
	bs_router "github.com/3zheng/railgun/protodefine/router"
	bs_tcp "github.com/3zheng/railgun/protodefine/tcpnet"

	_ "github.com/go-sql-driver/mysql"
	proto "github.com/golang/protobuf/proto"
)

type NetAgent struct {
	IpAddress string
}

var netAgentList []NetAgent

func (*NetAgent) SendMsg(req *bs_tcp.TCPTransferMsg) {
	var connId uint64 = req.Base.ConnId
	sess := ListenManager.GetSessionByConnId(connId)
	if sess == nil { //判断是否为空
		return
	}
	//往MsgWriteCh里写req
	select {
	case sess.MsgWriteCh <- req:
	case <-time.After(5 * time.Second):
		fmt.Println("严重事件，sess.MsgWriteCh已经阻塞超时5秒, connId =", connId)
	}
}

//创建一个NetAgent,ipAdd是个带端口的ip地址，如果是监听一个端口使用0.0.0.0:port
//NetAgent是TcpManager的映射，在CreateNetAgent并不和TcpManager联系起来，而是在SingleMsgPool的InitAndRun来创建TcpManager的TCP端口监听
func CreateNetAgent(ipAdd string) *NetAgent {
	for _, v := range netAgentList {
		if v.IpAddress == ipAdd {
			fmt.Println("已存在相同IP地址和端口,返回空指针")
			return nil
		}
	}
	agent := new(NetAgent)
	agent.IpAddress = ipAdd
	netAgentList = append(netAgentList, agent)
	return agent
}

type RouterAgent struct {
	IpAddress   string
	DialSession *DialManager.ConnectionSession
}

func CreateRouterAgent(ipAdd string) *RouterAgent {
	agent := new(RouterAgent)
	agent.IpAddress = ipAdd
	return agent
}

//启动运行RouterAgent
func (this *RouterAgent) RunRouterAgent(RouterToLogicChannel chan proto.Message, myAppId uint32, myAppType uint32) {
	fmt.Println("RunRouterAgent() myAppId=", myAppId, "myAppType=", myAppType)
	isClosed := false
	this.DialSession = DialManager.CreateClient(this.IpAddress, RouterToLogicChannel)
	for {
		//在这个router session关闭后，需要不停的尝试重新连接
		for isClosed || this.DialSession == nil {
			select {
			case <-time.After(700 * time.Millisecond): //每700毫秒尝试重新连接一次
			}
			this.DialSession = DialManager.CreateClient(this.IpAddress, RouterToLogicChannel)
			if this.DialSession != nil {
				//重连成功了
				fmt.Println("router Agent重连成功了")
				//这里用break是不能跳出for循环的，只能跳出当前的select，所以只能isClosed = false
				isClosed = false
			}
		}
		//连上router后向router发送注册报文
		req := new(bs_router.RegisterAppReq)
		bs_proto.SetBaseKindAndSubId(req)
		req.AppType = myAppType
		req.AppId = myAppId
		buff, err := proto.Marshal(req)
		if err != nil {
			fmt.Println("序列化RegisterAppReq出错")
		}
		tcpMsg := new(bs_tcp.TCPTransferMsg)
		bs_proto.SetBaseKindAndSubId(tcpMsg)
		tcpMsg.Data = buff
		tcpMsg.DataKindId = uint32(bs_types.CMDKindId_IDKindRouter)
		tcpMsg.DataSubId = uint32(bs_router.CMDID_Router_IDRegisterAppReq)
		this.SendMsg(tcpMsg)
		fmt.Println("向router app发送了注册请求")
		//阻塞在这里直到session关闭
		select {
		case v := <-this.DialSession.Quit:
			if v {
				fmt.Println("router agent的连接已关闭")
				isClosed = true
				this.DialSession = nil
			}
		}
	}
}

func (this *RouterAgent) SendMsg(req *bs_tcp.TCPTransferMsg) {
	var connId uint64 = req.Base.ConnId
	if this.DialSession == nil { //判断是否为空
		return
	}
	//往MsgWriteCh里写req
	select {
	case this.DialSession.MsgWriteCh <- req:
	case <-time.After(5 * time.Second):
		fmt.Println("严重事件，DialSession.MsgWriteCh已经阻塞超时5秒, connId =", connId)
	}

}

type DBReadInfo struct {
	TheRows    *sql.Rows      //如果是读操作，这里是返回的数据集
	TheColumns map[string]int //返回的列名,key是列名，value是表示这个是第几列，在返回的时候好查找
	ArrValues  [][]string     //返回的结果值
	RowNum     int            //行数，其实就是len(ArrValues)
}

func (this *DBReadInfo) Clear() {
	this.TheRows = nil
	this.TheColumns = nil
	this.ArrValues = nil
	this.RowNum = 0
}

type DBWriteInfo struct {
	LastId       int64 //自增列的当前值
	AffectedRows int64 //写操作影响的行数
}

//数据库的映射
type CADODatabase struct {
	DBSourceString string      //mysql的数据库连接串	格式"root:123456@tcp(localhost:3306)/sns?charset=utf8"
	Err            error       //错误
	TheDB          *sql.DB     //数据库对象
	ReadInfo       DBReadInfo  //如果是select的读操作，相关返回结果存这里
	WriteInfo      DBWriteInfo //如果是update或insert的写操作，相关返回结果存这里
}

func CreateADODatabase(DBSourceString string) *CADODatabase {
	DBAgent := new(CADODatabase)
	DBAgent.DBSourceString = DBSourceString
	return DBAgent
}

func (this *CADODatabase) InitDB() {
	db, err := sql.Open("mysql", this.DBSourceString)
	if err != nil {
		fmt.Println("初始化失败，error=", err.Error())
		panic(err.Error())
		this.Err = err
	}
	fmt.Println("数据库连接初始化成功")
	this.TheDB = db
}

//从数据库里读数据
func (this *CADODatabase) ReadFromDB(sqlExpress string) {
	//在读数据前先clear DBReadInfo
	this.ReadInfo.Clear()
	rows, err := this.TheDB.Query(sqlExpress)
	if err != nil {
		fmt.Println("读数据库出错,sql语句=", sqlExpress, "error=", err.Error())
		this.Err = err
		return
	}
	this.ReadInfo.TheRows = rows
	columns, err := rows.Columns()
	if err != nil {
		fmt.Println("取列名出错,sql语句=", sqlExpress, "error=", err.Error())
		this.Err = err
		return
	}

	this.ReadInfo.TheColumns = make(map[string]int)
	//记录列名及其对应的数组下标
	for i, v := range columns {
		this.ReadInfo.TheColumns[v] = i
	}
	colNum := len(columns) //列的数量
	// Make a slice for the values
	values := make([]sql.RawBytes, colNum)
	// rows.Scan wants '[]interface{}' as an argument, so we must copy the
	// references into such a slice
	// See http://code.google.com/p/go-wiki/wiki/InterfaceSlice for details
	scanArgs := make([]interface{}, colNum)
	for i := range values {
		scanArgs[i] = &values[i]
	}

	// Fetch rows
	for rows.Next() {
		// get RawBytes from data
		err = rows.Scan(scanArgs...)
		if err != nil {
			fmt.Println("从rows里取数据出错,sql语句=", sqlExpress, "error=", err.Error())
			this.Err = err
			return
		}

		// Now do something with the data.
		// Here we just print each column as a string.
		var value string
		rowValue := make([]string, colNum)
		for i, col := range values {
			// Here we can check if the value is nil (NULL value)
			if col == nil {
				value = "NULL"
			} else {
				value = string(col)
			}
			rowValue[i] = value
		}
		//把取出来的值放进二维数组中
		this.ReadInfo.ArrValues = append(this.ReadInfo.ArrValues, rowValue)
	}
	this.ReadInfo.RowNum = len(this.ReadInfo.ArrValues)
}

//向数据库写数据
func (this *CADODatabase) WriteToDB(sqlExpress string) {
	result, err := this.TheDB.Exec(sqlExpress)
	if err != nil {
		fmt.Println("写数据库出错,sql语句=", sqlExpress, "error=", err.Error())
		this.Err = err
		return
	}
	this.WriteInfo.LastId, _ = result.LastInsertId()
	this.WriteInfo.AffectedRows, _ = result.RowsAffected()
}

//在ReadFromDB从结果集里获取数据,结果放入value中，所以value要传地址。rowId从0开始为第一行
//成功返回true，失败返回false
func (this *CADODatabase) GetValueByRowIdAndColName(rowId int, colName string, value interface{}) bool {
	//先根据列名获得列的数组下标
	colId, ok := this.ReadInfo.TheColumns[colName]
	if !ok {
		fmt.Println("不存在的列名,colName=", colName)
		return false
	}
	if rowId >= this.ReadInfo.RowNum {
		fmt.Println("行号越界,rowId=", rowId, ",RowNum=", this.ReadInfo.RowNum)
		return false
	}
	//进行数据转换
	strValue := this.ReadInfo.ArrValues[rowId][colId]
	var err error
	switch data := value.(type) {
	case *int:
		*data, err = strconv.Atoi(strValue)
	case *int32:
		var v int64
		v, err = strconv.ParseInt(strValue, 10, 32)
		*data = int32(v)
	case *int64:
		*data, err = strconv.ParseInt(strValue, 10, 64)
	case *uint:
		var v uint64
		v, err = strconv.ParseUint(strValue, 10, 64)
		*data = uint(v)
	case *uint32:
		var v uint64
		v, err = strconv.ParseUint(strValue, 10, 32)
		*data = uint32(v)
	case *uint64:
		*data, err = strconv.ParseUint(strValue, 10, 64)
	case *float32:
		var v float64
		v, err = strconv.ParseFloat(strValue, 32)
		*data = float32(v)
	case *float64:
		*data, err = strconv.ParseFloat(strValue, 64)
	case *string:
		*data = strValue
	}
	if err != nil {
		fmt.Println("数据转换出错")
		return false
	}

	return true
}

type ILogicProcess interface {
	Init(myPool *SingleMsgPool) bool                       //初始化
	ProcessReq(req proto.Message, pDatabase *CADODatabase) //处理来自所绑定的SingleMsgPool发送的报文
	OnPulse(ms uint64)                                     //定时函数，每隔200ms调用一次
}
