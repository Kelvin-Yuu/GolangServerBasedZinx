package main

import (
	"fmt"
	"zinx_server/mmo_game/apis"
	"zinx_server/mmo_game/core"
	"zinx_server/zinx/ziface"
	"zinx_server/zinx/znet"
)

// 当前客户端建立连接之后的hook函数
func OnConnectionAdd(conn ziface.IConnection) {
	//创建一个Player对象
	player := core.NewPlayer(conn)

	//给客户端发送MsgId:1的消息：同步当前Player的ID给客户端
	player.SyncPid()

	//给客户端发送MsgId:200的对象：同步当前Player的初始位置给客户端
	player.BroadCastStartPosition()

	//将当前新上线的玩家添加到WorldManager中
	core.WorldMgrObj.AddPlayer(player)

	//将该连接绑定一个玩家ID pid的属性
	conn.SetProperty("pid", player.Pid)

	//同步周边玩家，告知他们当前玩家已经上线，广播当前玩家的位置信息
	player.SyncSurrounding()

	fmt.Println("====> Player pid =", player.Pid, "is arrived. <====")
}

// 给当前链接断开之前触发的hook函数
func OnConnectionLost(conn ziface.IConnection) {
	//创建一个Player对象
	pid, _ := conn.GetProperty("pid")

	player := core.WorldMgrObj.GetPlayerByPid(pid.(int32))

	//触发玩家下线的业务
	player.Offline()

	fmt.Println("====> Player pid =", pid, "is offlined.")

}

func main() {
	//创建server句柄
	s := znet.NewServer("MMO Game Server")

	//连接创建和销毁Hook函数
	s.SetOnConnStart(OnConnectionAdd)
	s.SetOnConnStop(OnConnectionLost)

	//注册路由
	s.AddRouter(2, &apis.WorldChatApi{})
	s.AddRouter(3, &apis.MoveApi{})

	//启动服务
	s.Server()
}
