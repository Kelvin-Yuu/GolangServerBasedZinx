package main

import (
	"fmt"
	"net"
	"time"
)

/*
模拟客户端
*/
func main() {
	fmt.Println("Client Start...")
	time.Sleep(1 * time.Second)

	//1 直接链接远程服务器，得到一个conn链接
	conn, err := net.Dial("tcp", "127.0.0.1:8999")
	if err != nil {
		fmt.Println("Client start err, exit!", err)
		return
	}

	//2 链接调用Write写数据
	for {
		_, err := conn.Write([]byte("Hello ZinxV0.2"))
		if err != nil {
			fmt.Println("conn Write err: ", err)
			return
		}

		buf := make([]byte, 512)
		cnt, err := conn.Read(buf)
		if err != nil {
			fmt.Println("read buf error: ", err)
			return
		}
		fmt.Printf("Server call back: %s, cnt = %d\n", buf, cnt)

		//cpu阻塞
		time.Sleep(1 * time.Second)
	}
}
