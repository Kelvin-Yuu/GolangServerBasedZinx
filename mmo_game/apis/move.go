package apis

import (
	"fmt"
	"zinx_server/mmo_game/core"
	"zinx_server/mmo_game/pb"
	"zinx_server/zinx/ziface"
	"zinx_server/zinx/znet"

	"google.golang.org/protobuf/proto"
)

// 玩家移动
type MoveApi struct {
	znet.BaseRouter
}

func (m *MoveApi) Handle(request ziface.IRequest) {
	//解析客户端传递过来的proto_msg
	proto_msg := &pb.Position{}
	if err := proto.Unmarshal(request.GetData(), proto_msg); err != nil {
		fmt.Println("Move: Position Unmarshal Error!", err)
		return
	}

	//得到当前发送位置的是哪个玩家
	pid, err := request.GetConnection().GetProperty("pid")
	if err != nil {
		fmt.Println("GetProperty pid Error!", err)
		return
	}
	fmt.Printf("Player pid=%d, move(%f,%f,%f,%f)\n", pid, proto_msg.X, proto_msg.Y, proto_msg.Z, proto_msg.V)

	//给其他玩家进行当前玩家的位置信息广播
	player := core.WorldMgrObj.GetPlayerByPid(pid.(int32))
	//广播并更新当前玩家的坐标
	player.UpdatePos(proto_msg.X, proto_msg.Y, proto_msg.Z, proto_msg.V)

}
