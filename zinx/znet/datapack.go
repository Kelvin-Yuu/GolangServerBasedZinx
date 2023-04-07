package znet

import (
	"bytes"
	"encoding/binary"
	"errors"
	"zinx_server/zinx/utils"
	"zinx_server/zinx/ziface"
)

/*
封包、拆包的具体模块
*/
type DataPack struct {
}

// 拆包封包实例的初始化方法
func NewDataPack() *DataPack {
	return &DataPack{}
}

// 获取包头的长度方法
func (dp *DataPack) GetHeadLen() uint32 {
	//Datalan uint32(4字节) + ID uint32(4字节)、
	return 8
}

// 封包方法
// dataLen|msgId|data
func (dp *DataPack) Pack(msg ziface.IMessage) ([]byte, error) {
	//创建一个存放bytes字节的缓冲
	dataBuff := bytes.NewBuffer([]byte{})

	//将dataLen写进dataBuff中
	if err := binary.Write(dataBuff, binary.LittleEndian, msg.GetDataLen()); err != nil {
		return nil, err
	}

	//将MsgId写进dataBuff中
	if err := binary.Write(dataBuff, binary.LittleEndian, msg.GetMsgId()); err != nil {
		return nil, err
	}

	//将data数据写进dataBuff中
	if err := binary.Write(dataBuff, binary.LittleEndian, msg.GetData()); err != nil {
		return nil, err
	}

	return dataBuff.Bytes(), nil
}

// 拆包方式
// 将包的Head信息读出来，之后再根据Head信息里的dataLen，再进行一次读操作
func (dp *DataPack) UnPack(binaryData []byte) (ziface.IMessage, error) {

	//创建一个从输入二进制数据的ioReader
	dataBuff := bytes.NewReader(binaryData)

	//只解压Head信息，得到dataLen和MsgID
	msg := &Message{}

	//读取dataLen
	if err := binary.Read(dataBuff, binary.LittleEndian, &msg.DataLen); err != nil {
		return nil, err
	}

	//读取MsgID
	if err := binary.Read(dataBuff, binary.LittleEndian, &msg.Id); err != nil {
		return nil, err
	}

	//判断dataLen是否已经超出了我们允许的最大包长度
	if utils.GlobalObject.MaxPackageSize > 0 && msg.DataLen > utils.GlobalObject.MaxPackageSize {
		return nil, errors.New("too Large msg data recv! ")
	}

	return msg, nil
}
