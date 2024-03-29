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

	err := request.GetConnection().SendMsg(request.GetMsgID()+200, []byte("ping...ping...ping"))
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

type HelloRouter struct {
	znet.BaseRouter
}

// Test Handle
func (hR *HelloRouter) Handle(request ziface.IRequest) {
	fmt.Println("Call Router Handle...")

	//先读取客户端的数据，再回写ping...ping...ping
	fmt.Println("recv from client: MsgID = ", request.GetMsgID(), ", data = ", string(request.GetData()))

	err := request.GetConnection().SendMsg(request.GetMsgID()+200, []byte("Hello! Welcome Zinx"))
	if err != nil {
		fmt.Println(err)
	}
}

// 创建链接之后执行hook函数
func DoConnectionBegin(conn ziface.IConnection) {
	fmt.Println("----> DoConnectionBegin is Called...")
	if err := conn.SendMsg(202, []byte("DoConnection Begin")); err != nil {
		fmt.Println(err)
	}
}

// 创建断开之前执行hook函数
func DoConnectionLost(conn ziface.IConnection) {
	fmt.Println("----> DoConnectionLost is Called...")
	fmt.Println("conn ID = ", conn.GetConnID(), " is Lost...")
}

func main() {
	//1 创建一个server句柄，使用Zinx的api
	s := znet.NewServer("[zinx V0.5]")

	//2 注册链接Hook函数
	s.SetOnConnStart(DoConnectionBegin)
	s.SetOnConnStop(DoConnectionLost)

	//3 给当前框架添加一个自定义router
	s.AddRouter(0, &PingRouter{})
	s.AddRouter(1, &HelloRouter{})

	//3 启动Server
	s.Serve()
}
