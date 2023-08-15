package ziface

import (
	"context"
	"github.com/gorilla/websocket"
	"net"
)

// 定义链接模块的抽象层
type IConnection interface {
	//启动链接 让当前的链接准备开始工作
	Start()

	//停止链接 结束当前链接的工作
	Stop()

	//获取当前链接的绑定socket conn
	GetConnection() net.Conn

	//从当前连接获取原始的socket TCPConn
	GetTCPConnection() net.Conn

	//从当前连接中获取原始的websocket连接
	GetWsConn() *websocket.Conn

	//获取当前链接模块的链接ID
	GetConnID() uint64

	//获取当前字符串连接ID
	GetConnIdStr() string

	//获取消息处理器
	GetMsgHandler() IMsgHandle

	//获取workerid
	GetWorkerID() uint32

	//获取远程客户端的TCP状态（IP,Port)
	RemoteAddr() net.Addr
	RemoteAddrString() string

	//获取本地服务器的TCP状态（IP,Port）
	LocalAddr() net.Addr
	LocalAddrString() string

	//将数据直接发送到远程TCP客户端（无缓冲）
	Send(data []byte) error

	//将数据发送到稍后要发送到远程TCP客户端的消息队列
	SendToQueue(data []byte) error

	//直接将Message数据发送数据给远程的TCP客户端(无缓冲)
	SendMsg(msgId uint32, data []byte) error

	//直接将Message数据发送给远程的TCP客户端(有缓冲)
	SendBuffMsg(msgId uint32, data []byte) error //添加带缓冲发送消息接口

	//设置链接属性
	SetProperty(key string, value interface{})

	//获取链接属性
	GetProperty(key string) (interface{}, error)

	//移除链接属性
	RemoveProperty(key string)

	//判断当前链接是否存活
	IsAlive() bool

	//设置心跳检测器
	SetHeartBeat(checker IHeartbeatChecker)

	//返回ctx，用于用户自定义的go程获取连接退出状态
	Context() context.Context
}
