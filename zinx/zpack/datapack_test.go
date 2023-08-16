package zpack

import (
	"fmt"
	"io"
	"net"
	"testing"
	"time"
	"zinx_server/zinx/ziface"
)

func TestDataPack(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:7777")
	if err != nil {
		fmt.Println("server listen err:", err)
		return
	}
	// server goroutine
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				fmt.Println("server accept err:", err)
				return
			}

			go func(conn net.Conn) {
				dp := Factory().NewPack(ziface.ZinxDataPack)
				for {
					// 1. 读取包头
					headData := make([]byte, dp.GetHeadLen())
					_, err := io.ReadFull(conn, headData)
					if err != nil {
						fmt.Println("read head err:", err)
					}
					// 2. 解包包头
					msgHead, err := dp.Unpack(headData)
					if err != nil {
						fmt.Println("server unpack err:", err)
						return
					}

					if msgHead.GetDataLen() > 0 {
						// 3. msg有data, 再次读取
						msg := msgHead.(*Message)
						msg.Data = make([]byte, msg.GetDataLen())

						// 4. 根据dataLen读取字节流
						_, err := io.ReadFull(conn, msg.Data)
						if err != nil {
							fmt.Println("server unpack data err:", err)
							return
						}
						fmt.Println("==> Recv Msg: ID=", msg.ID, ", len=", msg.DataLen, ", data=", string(msg.Data))
					}
				}
			}(conn)
		}
	}()

	// client goroutine
	go func() {
		for {
			conn, err := net.Dial("tcp", "127.0.0.1:7777")
			if err != nil {
				fmt.Println("client dial err:", err)
				return
			}

			dp := Factory().NewPack(ziface.ZinxDataPack)

			// Package msg1
			msg1 := &Message{
				ID:      0,
				DataLen: 5,
				Data:    []byte{'h', 'e', 'l', 'l', 'o'},
			}
			sendData1, err := dp.Pack(msg1)
			if err != nil {
				fmt.Println("client pack msg1 err:", err)
				return
			}
			// Package msg2.
			msg2 := &Message{
				ID:      1,
				DataLen: 7,
				Data:    []byte{'w', 'o', 'r', 'l', 'd', '!', '!'},
			}
			sendData2, err := dp.Pack(msg2)
			if err != nil {
				fmt.Println("client temp msg2 err:", err)
				return
			}

			sendData1 = append(sendData1, sendData2...)

			conn.Write(sendData1)

		}
	}()

	select {
	case <-time.After(time.Second):
		return
	}

}
