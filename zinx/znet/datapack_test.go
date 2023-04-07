package znet

import (
	"fmt"
	"io"
	"net"
	"testing"
)

// 只是负责测试dataPack拆包，封包的单元测试
func TestDataPack(t *testing.T) {
	/*
		模拟的服务器
	*/
	//1 创建socketTCP
	listenner, err := net.Listen("tcp", "0.0.0.0:8999")
	if err != nil {
		fmt.Println("server listen err, ", err)
		return
	}

	//创建一个go承载 负责从客户端处理业务
	go func() {
		//2 从客户端读取数据，进行拆包处理
		for {
			conn, err := listenner.Accept()
			if err != nil {
				fmt.Println("serve accept, error, ", err)
			}

			go func(conn net.Conn) {
				//处理客户端的请求
				//----> 拆包的过程 <----
				//定义一个拆包对象dp
				dp := NewDataPack()
				for {
					//1 第一次从conn读，把包的head读出来
					headData := make([]byte, dp.GetHeadLen())
					_, err := io.ReadFull(conn, headData)
					if err != nil {
						fmt.Println("read head error, ", err)
						break
					}
					msgHead, err := dp.UnPack(headData)
					if err != nil {
						fmt.Println("server unpack err, ", err)
					}

					if msgHead.GetDataLen() > 0 {
						//Msg是有数据的，需要进行第二次读取
						//2 第二次从conn读，根据head的dataLen读取data内容
						msg := msgHead.(*Message)
						msg.Data = make([]byte, msg.GetDataLen())

						//根据dataLen再次从io流中读取data
						_, err := io.ReadFull(conn, msg.Data)
						if err != nil {
							fmt.Println("server unpack data err: ", err)
							return
						}

						//完整的一个消息已经读取完毕
						fmt.Println("Recv MsgID: ", msg.Id, ", DataLen = ", msg.DataLen, ", data = ", string(msg.Data))
					}

				}
			}(conn)
		}
	}()

	/*
		模拟的客户端
	*/
	conn, err := net.Dial("tcp", "127.0.0.1:8999")
	if err != nil {
		fmt.Println("client dial err: ", err)
		return
	}

	//创建一个封包对象dp
	dp := NewDataPack()

	//模拟粘包过程，封装两个msg一同发送
	//封装第一个msg1包
	strMsg := "zinxTest"
	msg1 := &Message{
		Id:      1,
		DataLen: uint32(len(strMsg)),
		Data:    []byte(strMsg),
	}
	sendData1, err := dp.Pack(msg1)
	if err != nil {
		fmt.Println("client pack msg1 error, ", err)
		return
	}
	//封装第二个msg2包
	strMsg = "你好!!!"
	msg2 := &Message{
		Id:      2,
		DataLen: uint32(len(strMsg)),
		Data:    []byte(strMsg),
	}
	sendData2, err := dp.Pack(msg2)
	if err != nil {
		fmt.Println("client pack msg2 error, ", err)
		return
	}
	//将两个包粘在一起
	sendData1 = append(sendData1, sendData2...)

	//一次性将两个包发送给server
	conn.Write(sendData1)

	//客户端阻塞
	select {}
}
