package zpack

import (
	"bytes"
	"encoding/binary"
	"errors"
	"zinx_server/zinx/zconf"
	"zinx_server/zinx/ziface"
)

var defaultHeaderLen uint32 = 8

type DataPack struct{}

// 封包拆包实例初始化方法
func NewDataPack() ziface.IDataPack {
	return &DataPack{}
}

// 获取包头长度
func (dp *DataPack) GetHeadLen() uint32 {
	return defaultHeaderLen
}

// 封包方法，压缩数据
func (dp *DataPack) Pack(msg ziface.IMessage) ([]byte, error) {
	dataBuff := bytes.NewBuffer([]byte{})

	// Write the message ID
	if err := binary.Write(dataBuff, binary.BigEndian, msg.GetMsgID()); err != nil {
		return nil, err
	}

	// Write the data length
	if err := binary.Write(dataBuff, binary.BigEndian, msg.GetDataLen()); err != nil {
		return nil, err
	}

	// Write the data
	if err := binary.Write(dataBuff, binary.BigEndian, msg.GetData()); err != nil {
		return nil, err
	}

	return dataBuff.Bytes(), nil
}

// 拆包方法，解压数据
func (dp *DataPack) UnPack(binaryData []byte) (ziface.IMessage, error) {
	dataBuff := bytes.NewReader(binaryData)
	msg := &Message{}

	// Read the data length
	if err := binary.Read(dataBuff, binary.BigEndian, &msg.ID); err != nil {
		return nil, err
	}

	// Read the message ID
	if err := binary.Read(dataBuff, binary.BigEndian, &msg.DataLen); err != nil {
		return nil, err
	}

	// 判断dataLen的长度是否超出我们允许的最大包长度
	if zconf.GlobalObject.MaxPacketSize > 0 && msg.GetDataLen() > zconf.GlobalObject.MaxPacketSize {
		return nil, errors.New("Too large msg data received! ")

	}

	// 这里只需要把head的数据拆包出来就可以了，然后再通过head的长度，再从conn读取一次数据
	return msg, nil
}
