package apis

import (
	"fmt"
	"zinx_server/mmo_game/core"
	"zinx_server/mmo_game/pb"
	"zinx_server/zinx/ziface"
	"zinx_server/zinx/znet"

	"google.golang.org/protobuf/proto"
)

// 世界聊天 路由业务
type WorldChatApi struct {
	znet.BaseRouter
}

func (wc *WorldChatApi) Handle(request ziface.IRequest) {
	//1 解析客户端传递进来的proto协议
	proto_msg := &pb.Talk{}
	err := proto.Unmarshal(request.GetData(), proto_msg)
	if err != nil {
		fmt.Println("TalkMsg Unmarshal Error!", err)
		return
	}

	//2 获取当前的聊天数据发送者
	pid, err := request.GetConnection().GetProperty("pid")
	if err != nil {
		fmt.Println("Request Get Pid Error!", err)
		return
	}

	//3 根据pid得到对应的player对象
	player := core.WorldMgrObj.GetPlayerByPid(pid.(int32))

	//4 将这个消息广播给其他全部在线的玩家
	player.Talk(proto_msg.Content)
}
