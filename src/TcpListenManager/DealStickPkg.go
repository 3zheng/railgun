package ListenManager

import (
	"bytes"
	"encoding/binary"
)

//接收消息要处理粘包的解包函数
//pkg是ConnectionSession的cache，如果根据头4个字节获取的报文长度大于当前pkg的len-4的长度，那么说明还有后续消息没有read，还需要继续等待。这时返回nil []byte
//反之如果长度小于等于当前pkg的len-4的长度说明，说明已经获取了一个完整的消息，就把头四个字节去掉，然后把实际报文return出去
func DecodePackage(pkg *bytes.Buffer, recvData []byte) []byte {
	//先把接收的消息写入pkg
	pkg.Write(recvData)
	if pkg.Len() < 4 {
		//不足4个字节直接返回
		return nil
	}
	//取头四个字节
	//这里binary.Read不能直接使用pkg，因为如果pkg剩余长度小于length指示的长度，那么pkg就回不到前四个字节了
	//所以先把pkg头四个直接写入临时变量tmppkg中
	var length int32 = 0
	tmppkg := new(bytes.Buffer)
	tmppkg.Write(pkg.Bytes()[:4])
	binary.Read(tmppkg, binary.BigEndian, &length)
	if int(length)+4 > pkg.Len() { //+4是因为pkg的报文头还在
		//不足所需的报文长度直接返回
		return nil
	} else {
		//先读取4字节
		binary.Read(pkg, binary.BigEndian, &length)
		buff := make([]byte, length) //构造长度为length的buff
		binary.Read(pkg, binary.BigEndian, buff)
		return buff
	}

	return nil
}

//发消息时调用的组包函数
func EncodePackage(pkg *bytes.Buffer, data []byte) error {
	var length int32 = int32(len(data))
	// 先把表示消息长度的4个字节int32写入消息头,而且TCP流得是BigEndian大端字节序
	err := binary.Write(pkg, binary.BigEndian, length)
	if err != nil {
		return err
	}
	// 写入消息实体
	err = binary.Write(pkg, binary.BigEndian, data)
	if err != nil {
		return err
	}

	return nil
}
