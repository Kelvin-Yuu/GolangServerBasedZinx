package main

import (
	"fmt"
	"zinx_server/zinx/ziface"
	"zinx_server/zinx/znet"
)

/*
	基于Zinx框架开发的服务器端应用程序
*/

// ping test 自定义路由
type PingRouter struct {
	znet.BaseRouter
}

// // Test PreRouter
// func (pR *PingRouter) PreHandle(request ziface.IRequest) {
// 	fmt.Println("Call Router PreHandle...")
// 	_, err := request.GetConnection().GetConnection().Write([]byte("before ping...\n"))
// 	if err != nil {
// 		fmt.Println("call back before Ping error: ", err)
// 	}
// }

// Test Handle
func (pR *PingRouter) Handle(request ziface.IRequest) {
	fmt.Println("Call Router Handle...")

	//先读取客户端的数据，再回写ping...ping...ping
	fmt.Println("recv from client: MsgID = ", request.GetMsgID(), ", data = ", string(request.GetData()))

	err := request.GetConnection().SendMsg(request.GetMsgID(), []byte("ping...ping...ping"))
	if err != nil {
		fmt.Println(err)
	}
}

// // Test PostHandle
// func (pR *PingRouter) PostHandle(request ziface.IRequest) {
// 	fmt.Println("Call Router PostHandle...")
// 	_, err := request.GetConnection().GetConnection().Write([]byte("After ping...\n"))
// 	if err != nil {
// 		fmt.Println("call back after ping error: ", err)
// 	}
// }

func main() {
	//1 创建一个server句柄，使用Zinx的api
	s := znet.NewServer("[zinx V0.5]")

	//2 给当前框架添加一个自定义router
	s.AddRouter(&PingRouter{})

	//3 启动Server
	s.Serve()
}
