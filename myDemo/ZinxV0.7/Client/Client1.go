package main

import (
	"fmt"
	"io"
	"net"
	"time"
	"zinx_server/zinx/znet"
)

/*
模拟客户端
*/
func main() {
	fmt.Println("Client Start...")

	time.Sleep(1 * time.Second)

	//1 直接链接远程服务器，得到一个conn链接
	conn, err := net.Dial("tcp", "127.0.0.1:7777")
	if err != nil {
		fmt.Println("Client start err, exit!", err)
		return
	}

	//2 链接调用Write写数据
	for {
		//发送封包的message消息
		dp := znet.NewDataPack()

		binaryMsg, err := dp.Pack(znet.NewMsgPackage(1, []byte("ZinxV0.5 Client1 Test Message")))

		if err != nil {
			fmt.Println("Pack error", err)
			break
		}
		if _, err := conn.Write(binaryMsg); err != nil {
			fmt.Println("write error", err)
			break
		}

		//服务器会回复一个message数据，MsgID:1 Data:ping...ping...ping

		//先读取流中的head部分，得到ID和DataLen
		binaryHead := make([]byte, dp.GetHeadLen())
		if _, err := io.ReadFull(conn, binaryHead); err != nil {
			fmt.Println("read head error, ", err)
			break
		}
		//将二进制的head拆包到msg结构体中
		msgHead, err := dp.UnPack(binaryHead)
		if err != nil {
			fmt.Println("Client unpack msgHead Error!", err)
		}
		if msgHead.GetDataLen() > 0 {
			//再根据dataLen进行二次读取，将data读取出来
			msg := msgHead.(*znet.Message)
			msg.Data = make([]byte, msg.GetDataLen())

			if _, err := io.ReadFull(conn, msg.Data); err != nil {
				fmt.Println("read msg data error!", err)
				return
			}
			fmt.Println("----> Recv From Server Msg: ID = ", msg.Id, ", Len = ", msg.DataLen, ", Data = ", string(msg.Data))
		}
		// cpu阻塞
		time.Sleep(time.Second)
	}
}
