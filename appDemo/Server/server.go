package main

import (
	"fmt"
	"zinx_server/appDemo/Api_MMO_GAME/api"
	"zinx_server/appDemo/Api_MMO_GAME/core"
	"zinx_server/zinx/zdecoder"
	"zinx_server/zinx/ziface"
	"zinx_server/zinx/znet"
	"zinx_server/zinx/zpack"
)

func OnConnectionAdd(conn ziface.IConnection) {
	fmt.Println("=====> [Server] OnConnectionAdd is Called ...")

	//创建一个玩家
	player := core.NewPlayer(conn)

	//同步当前的Player给客户端  消息MsgId: 1
	player.SyncPID()

	//同步当前的Player的初始化坐标信息给客户端，MsgId:200
	player.BroadCastStartPosition()

	//将当前新上线Player添加到worldManager中
	core.WorldMgrObj.AddPlayer(player)

	//将该连接绑定属性PID
	conn.SetProperty("pID", player.PID)

	//同步周边Player上线信息，与现实周边Player信息
	player.SyncSurrounding()

	fmt.Println("=====> [Server] Player pIDID = ", player.PID, " arrived ====")
}

func OnConnectionLost(conn ziface.IConnection) {
	pID, _ := conn.GetProperty("pID")

	var playerID int32
	if pID != nil {
		playerID = pID.(int32)
	}

	//根据pID获取对应的Player对象
	player := core.WorldMgrObj.GetPlayerByPID(playerID)

	//触发玩家下线业务
	if player != nil {
		player.LostConnection()
	}
	fmt.Println("====> [Server] Player ", playerID, " left =====")
}

func main() {
	s := znet.NewServer()

	s.SetOnConnStart(OnConnectionAdd)
	s.SetOnConnStop(OnConnectionLost)

	s.AddRouter(2, &api.WorldChatApi{})
	s.AddRouter(3, &api.MoveApi{})

	s.SetDecoder(zdecoder.NewLTV_Little_Decoder())
	s.SetPacket(zpack.NewDataPackLtv())

	s.Serve()

}
