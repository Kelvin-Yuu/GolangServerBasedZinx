package core

import (
	"fmt"
	"math/rand"
	"sync"
	"zinx_server/mmo_game/pb"
	"zinx_server/zinx/ziface"

	"google.golang.org/protobuf/proto"
)

// 玩家对象
type Player struct {
	Pid  int32              //玩家对象
	Conn ziface.IConnection //当前玩家的连接（用于和客户端的连接）
	X    float32            //平面的x坐标
	Y    float32            //高度
	Z    float32            //平面的y坐标
	V    float32            //旋转的0-360角度
}

// PlayerID生成器
var PidGen int32 = 1  //用来生产玩家ID的计数器
var IdLock sync.Mutex //保护PidGen的mutex

// 创建玩家的方法
func NewPlayer(conn ziface.IConnection) *Player {
	//生成一个玩家ID
	IdLock.Lock()
	id := PidGen
	PidGen++
	IdLock.Unlock()

	return &Player{
		Pid:  id,
		Conn: conn,
		X:    float32(160 + rand.Intn(10)),
		Y:    0,
		Z:    float32(140 + rand.Intn(20)),
		V:    0,
	}
}

/*
提供一个发送给客户端消息的方法
主要是将pb的protobuf数据序列化之后，再调用zinx的SendMsg方法
*/
func (p *Player) SendMsg(msgId uint32, data proto.Message) {
	//将proto Msg结构体序列化，转换成二进制
	msg, err := proto.Marshal(data)
	if err != nil {
		fmt.Println("marshal msg err:", err)
		return
	}

	//将二进制文件 通过zinx框架的sendMsg将数据发送给客户端
	if p.Conn == nil {
		fmt.Println("Connection in player is nil!")
		return
	}

	if err := p.Conn.SendMsg(msgId, msg); err != nil {
		fmt.Println("Player SendMsg Error:", err)
		return
	}

}

// 告知客户端玩家Pid，同步已经生成的玩家pid给客户端
func (p *Player) SyncPid() {
	//组建MsgId:1 的proto数据
	proto_msg := &pb.SyncPid{
		Pid: p.Pid,
	}

	//将消息发送给客户端
	p.SendMsg(1, proto_msg)
}

// 广播玩家自己的出生地点
func (p *Player) BroadCastStartPosition() {
	//组建MsgId:200 的proto数据
	proto_msg := &pb.BroadCast{
		Pid: p.Pid,
		Tp:  2,
		Data: &pb.BroadCast_P{
			//Position
			P: &pb.Position{
				X: p.X,
				Y: p.Y,
				Z: p.Z,
				V: p.V,
			},
		},
	}

	//将消息发送给客户端
	p.SendMsg(200, proto_msg)
}

// 玩家广播世界聊天消息
func (p *Player) Talk(content string) {
	//1 组建MsgId:200 proto数据
	proto_msg := &pb.BroadCast{
		Pid: p.Pid,
		Tp:  1, //1-代表世界聊天
		Data: &pb.BroadCast_Content{
			Content: content,
		},
	}

	//2 得到当前世界所有的在线玩家
	players := WorldMgrObj.GetAllPlayers()

	//3 向所有的在线玩家发送消息
	for _, player := range players {
		player.SendMsg(200, proto_msg)
	}
}

// 同步玩家上线的位置消息
func (p *Player) SyncSurrounding() {
	//1 获取当前玩家周围的玩家有哪些（九宫格）
	pids := WorldMgrObj.AoiMgr.GetPIDsByPos(p.X, p.Z)
	players := make([]*Player, 0, len(pids))
	for _, pid := range pids {
		players = append(players, WorldMgrObj.GetPlayerByPid(int32(pid)))
	}

	//2 将当前玩家的位置信息通过MsgID:200 发送给周围玩家（让其他玩家看到自己）
	//2.1 组建MsgId:200 的proto Msg
	proto_msg := &pb.BroadCast{
		Pid: p.Pid,
		Tp:  2,
		Data: &pb.BroadCast_P{
			P: &pb.Position{
				X: p.X,
				Y: p.Y,
				Z: p.Z,
				V: p.V,
			},
		},
	}
	//2.2 周围的全部玩家都向各自的客户端发送200 proto_msg消息
	for _, player := range players {
		player.SendMsg(200, proto_msg)
	}

	//3 将周围的全部玩家的位置信息MsgID:202 发送给当前的玩家客户端（让自己看到其他玩家）
	//3.1 制作Message SyncPlayers 数据
	playersData := make([]*pb.Player, 0, len(players))
	for _, player := range players {
		p := &pb.Player{
			Pid: player.Pid,
			P: &pb.Position{
				X: player.X,
				Y: player.Y,
				Z: player.Z,
				V: player.V,
			},
		}
		playersData = append(playersData, p)
	}

	//3.2 封装SyncPlayer protobuf数据
	SyncPlayersMsg := &pb.SyncPlayers{
		Ps: playersData[:],
	}

	//3.3 给当前玩家发送需要显示周围的全部玩家数据
	p.SendMsg(202, SyncPlayersMsg)
}

// // 广播玩家位置移动
// func (p *Player) UpdatePos(x, y, z, v float32) {
// 	//更新玩家的位置信息
// 	p.X = x
// 	p.Y = y
// 	p.Z = z
// 	p.V = v

// 	//组建MsgId:200 的proto_msg 发送位置给周围玩家
// 	proto_msg := &pb.BroadCast{
// 		Pid: p.Pid,
// 		Tp:  4,
// 		Data: &pb.BroadCast_P{
// 			P: &pb.Position{
// 				X: p.X,
// 				Y: p.Y,
// 				Z: p.Z,
// 				V: p.V,
// 			},
// 		},
// 	}

// 	//获取当前玩家附近全部玩家
// 	players := p.GetSurroundingPlayers()

// 	//依次给每个玩家对应的客户端发送当前玩家位置更新的消息
// 	for _, player := range players {
// 		player.SendMsg(200, proto_msg)
// 	}

// }

func (p *Player) GetSurroundingPlayers() []*Player {
	//得到当前AOI九宫格内的所有玩家pid
	pids := WorldMgrObj.AoiMgr.GetPIDsByPos(p.X, p.Z)

	//将所有的pid对应的player放到players切片中
	players := make([]*Player, 0, len(pids))

	for _, pid := range pids {
		players = append(players, WorldMgrObj.GetPlayerByPid(int32(pid)))
	}

	return players
}

// 自行实现跨越格子的AOI处理
func (player *Player) UpdatePos(x, y, z, v float32) {
	// 处理跨越格子
	// 1. 获取玩家当前位置所处格子ID
	// 2. 获取玩家新位置所处格子ID
	// 3. 判断是否跨越格子 格子ID是否相同

	// 当前玩家所处格子ID
	curGridId := WorldMgrObj.AoiMgr.GetGIDByPos(player.X, player.Z)
	//fmt.Println("curGridId = ", curGridId)

	// 玩家新位置所处格子ID
	newGridId := WorldMgrObj.AoiMgr.GetGIDByPos(x, z)
	//fmt.Println("newGridId = ", newGridId)

	if curGridId != newGridId {
		// 说明跨越格子了，要处理新格子的视野，包括移除、新增其他玩家
		player.refreshAOI(x, y, z, v)
	} else {
		// 没有跨越格子则无需刷新视野
		// 更新玩家位置信息
		player.X = x
		player.Y = y
		player.Z = z
		player.V = v

		// 组建proto数据
		msg := &pb.BroadCast{
			Pid: player.Pid,
			Tp:  4,
			Data: &pb.BroadCast_P{
				P: &pb.Position{
					X: player.X,
					Y: player.Y,
					Z: player.Z,
					V: player.V,
				},
			},
		}

		// 获取玩家周围的其他玩家
		players := player.GetSurroundingPlayers()

		// 向周围的其他玩家广播移动消息
		for _, player := range players {
			player.SendMsg(200, msg)
		}
	}

}

// 刷新AOI视野
func (player *Player) refreshAOI(x, y, z, v float32) {
	// 1. 我离开旧的九宫格其他玩家的视野
	// 2. 旧的九宫格其他玩家消失在我的视野中
	// 3. 我出现在新的九宫格中的玩家视野中
	// 4. 新的九宫格的玩家出现在我的视野中

	// 获取旧九宫格所有玩家
	oldPlayerIdList := WorldMgrObj.AoiMgr.GetPIDsByPos(player.X, player.Z)

	// 获取新九宫格所有玩家
	newPlayerIdList := WorldMgrObj.AoiMgr.GetPIDsByPos(x, z)

	// 求两个玩家列表的格子的差集
	// 将old和new转换成Map
	oldMap := make(map[int]bool)
	for _, v := range oldPlayerIdList {
		oldMap[v] = true
	}

	newMap := make(map[int]bool)
	for _, v := range newPlayerIdList {
		newMap[v] = true
	}

	// 得到old数组中不在new数组中的元素
	var oldNotInNew []int
	for _, v := range oldPlayerIdList {
		if _, ok := newMap[v]; !ok {
			oldNotInNew = append(oldNotInNew, v)
		}
	}

	// 得到new数组中不在old数组中的元素
	var newNotInOld []int
	for _, v := range newPlayerIdList {
		if _, ok := oldMap[v]; !ok {
			newNotInOld = append(newNotInOld, v)
		}
	}

	fmt.Println("will remove playerId list: ", oldNotInNew)
	fmt.Println("will add playerId list: ", newNotInOld)

	// 获取需要移除视野的玩家实例
	removePlayers := make([]*Player, 0, len(oldNotInNew))

	for _, pid := range oldNotInNew {
		removePlayers = append(removePlayers, WorldMgrObj.GetPlayerByPid(int32(pid)))
	}
	// 1. 我离开旧的九宫格其他玩家的视野(广播玩家离开)
	msg := &pb.SyncPid{
		Pid: player.Pid,
	}
	for _, player := range removePlayers {
		player.SendMsg(201, msg)
	}

	// 2. 旧的九宫格其他玩家消失在我的视野中(移除旧视野中的其他玩家)
	for _, pid := range oldNotInNew {
		msg := &pb.SyncPid{
			Pid: int32(pid),
		}
		player.SendMsg(201, msg)
	}

	// 获取需要加入视野的玩家实例
	// addPlayers := make([]*Player, 0, len(newNotInOld))

	// for _, pid := range newNotInOld {
	//  addPlayers = append(addPlayers, WorldMgrObj.GetPlayerByPid(int32(pid)))
	// }

	// 3. 我出现在新的九宫格中的玩家视野中(广播玩家加入视野)
	// 4. 新的九宫格的玩家出现在我的视野中
	// 直接更新玩家最新位置信息，然后调用同步周围接口即可
	player.X = x
	player.Y = y
	player.Z = z
	player.V = v
	player.SyncSurrounding()

}

// 玩家下线
func (p *Player) Offline() {
	//得到当前玩家周边的九宫格内都有哪些玩家
	players := p.GetSurroundingPlayers()

	//给周围玩家广播MsgId:201 消息
	proto_msg := &pb.SyncPid{
		Pid: p.Pid,
	}

	for _, player := range players {
		player.SendMsg(201, proto_msg)
	}

	WorldMgrObj.AoiMgr.RemoveFromGrIDByPos(int(p.Pid), p.X, p.Z)
	WorldMgrObj.RemovePlayerByPid(p.Pid)
}
