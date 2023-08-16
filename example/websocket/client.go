package main

import (
	"fmt"
	"os"
	"os/signal"
	"time"
	"zinx_server/example/websocket/c_router"
	"zinx_server/zinx/ziface"
	"zinx_server/zinx/zlog"
	"zinx_server/zinx/znet"
)

type PositionClientRouter struct {
	znet.BaseRouter
}

func (this *PositionClientRouter) Handle(request ziface.IRequest) {

}

func business(conn ziface.IConnection) {
	for {
		err := conn.SendBuffMsg(1, []byte("ping ping ping ..."))
		if err != nil {
			fmt.Println(err)
		}
		time.Sleep(10 * time.Second)
	}
}

func DoClientConnectedBegin(conn ziface.IConnection) {
	//设置两个链接属性，在连接创建之后
	conn.SetProperty("Name", "KelvinU")

	go business(conn)
}

func main() {
	client := znet.NewWsClient("127.0.0.1", 9000)

	client.SetOnConnStart(DoClientConnectedBegin)

	client.AddRouter(2, &c_router.PingRouter{})
	client.AddRouter(3, &c_router.HelloRouter{})

	client.Start()

	select {
	case err := <-client.GetErrChan():
		zlog.Ins().ErrorF("client err:%v", err)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	sig := <-c
	fmt.Println("===exit===", sig)

	client.Stop()
	time.Sleep(time.Second * 2)
}
