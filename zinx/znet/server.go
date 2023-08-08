package znet

import (
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"net"
	"net/http"
	"time"
	"zinx_server/zinx/zconf"
	"zinx_server/zinx/ziface"
	"zinx_server/zinx/zlog"
)

// iServer的接口实现，定义一个Server的服务器模块
type Server struct {
	// Name of the server (服务器的名称)
	Name string
	//tcp4 or other
	IPVersion string
	// IP version (e.g. "tcp4") - 服务绑定的IP地址
	IP string
	// IP address the server is bound to (服务绑定的端口)
	Port int
	// 服务绑定的websocket 端口 (Websocket port the server is bound to)
	WsPort int
	// 服务绑定的kcp 端口 (kcp port the server is bound to)
	KcpPort int

	// Current server's message handler module, used to bind MsgID to corresponding processing methods
	// (当前Server的消息管理模块，用来绑定MsgID和对应的处理方法)
	msgHandler ziface.IMsgHandle

	// Routing mode (路由模式)
	RouterSlicesMode bool

	// Current server's connection manager (当前Server的链接管理器)
	ConnMgr ziface.IConnManager

	// Hook function called when a new connection is established
	// (该Server的连接创建时Hook函数)
	onConnStart func(conn ziface.IConnection)

	// Hook function called when a connection is terminated
	// (该Server的连接断开时的Hook函数)
	onConnStop func(conn ziface.IConnection)

	// Data packet encapsulation method
	// (数据报文封包方式)
	packet ziface.IDataPack

	// Asynchronous capture of connection closing status
	// (异步捕获链接关闭状态)
	exitChan chan struct{}

	//// Decoder for dealing with message fragmentation and reassembly
	//// (断粘包解码器)
	//decoder ziface.IDecoder

	// Heartbeat checker
	// (心跳检测器)
	hc ziface.IHeartbeatChecker

	// websocket
	upgrader *websocket.Upgrader

	// websocket connection authentication
	websocketAuth func(r *http.Request) error

	// connection id
	cID uint64
}

// 定义当前客户端链接的所绑定的handle api(目前这个handle是写死的，以后优化应该由用户自定义handle方法)
func CallBackToClient(conn *net.TCPConn, data []byte, cnt int) error {
	//回显的业务
	fmt.Println("[Conn Handle] CallBackToClient...")
	if _, err := conn.Write(data[:cnt]); err != nil {
		fmt.Println("writ back buf err: ", err)
		return errors.New("CallBackToClient error")
	}
	return nil
}

// 启动服务器
func (s *Server) Start() {
	zlog.Ins().InfoF("[Zinx] Server Name : %s, Server Listener at IP: %s, Port: %d\n",
		zconf.GlobalObject.Name,
		zconf.GlobalObject.Host,
		zconf.GlobalObject.TCPPort)
	zlog.Ins().InfoF("[Zinx] Version : %s, MaxConn: %d, MaxPackageSize: %d\n",
		zconf.GlobalObject.Version,
		zconf.GlobalObject.MaxConn,
		zconf.GlobalObject.MaxPacketSize)

	go func() {

		// 0 开启消息队列及Worker工作池
		s.msgHandler.StartWorkerPool()

		// 1 获取一个TCP的Addr
		addr, err := net.ResolveTCPAddr(s.IPVersion, fmt.Sprintf("%s:%d", s.IP, s.Port))
		if err != nil {
			fmt.Println("Resolve TCP addr error: ", err)
			return
		}

		// 2 监听服务器地址
		listenner, err := net.ListenTCP(s.IPVersion, addr)
		if err != nil {
			fmt.Println("Listen ", s.IPVersion, " error: ", err)
			return
		}

		fmt.Println("Start Zinx server successed, ", s.Name, ", Listenning now...")

		var cid uint32 = 0

		// 3 阻塞等待客户端连接，处理客户端连接业务
		for {
			//如果有客户端连接过来，阻塞会返回
			conn, err := listenner.AcceptTCP()
			if err != nil {
				fmt.Println("Accept error: ", err)
				continue
			}

			//设置最大连接个数的判断，如果超过最大连接，则关闭此新的连接
			if s.ConnMgr.Len() >= zconf.GlobalObject.MaxConn {
				//TODO 给客户端响应一个超出最大连接的错误包
				fmt.Println("Too many Connection, MaxConn=", zconf.GlobalObject.MaxConn)
				conn.Close()
				continue
			}

			//已经与客户端建立连接，做某些业务
			//将处理新链接的业务方法和conn进行绑定，得到我们的链接模块
			dealConn := NewConnection(s, conn, cid, s.msgHandler)
			cid++

			//启动当前的链接业务处理
			go dealConn.Start()

		}
	}()

}

// 停止服务器
func (s *Server) Stop() {
	//TODO 将一些服务器的资源、状态或者一些已经开辟的链接信息，进行停止或者回收
	fmt.Println("[STOP] Zinx server name ", s.Name)
	s.ConnMgr.ClearConn()
}

// 运行服务器
func (s *Server) Server() {
	//启动server的服务功能
	s.Start()

	//TODO 做一些启动服务器之后的额外业务

	//阻塞状态
	select {}
}

// 路由功能：给当前的服务注册一个路由方法，供客户端的链接处理使用
func (s *Server) AddRouter(msgID uint32, router ziface.IRouter) {
	s.msgHandler.AddRouter(msgID, router)
	fmt.Println("Add Router [MsgID =", msgID, "] Successed!")
}

func (s *Server) AddRouterSlices(msgId uint32, router ...ziface.RouterHandler) ziface.IRouterSlices {
	if !s.RouterSlicesMode {
		panic("Server RouterSlicesMode is FALSE!! ")
	}
	return s.msgHandler.AddRouterSlices(msgId, router...)
}

func (s *Server) Group(start, end uint32, Handlers ...ziface.RouterHandler) ziface.IGroupRouterSlices {
	if !s.RouterSlicesMode {
		panic("Server RouterSlicesMode is FALSE!! ")
	}
	return s.msgHandler.Group(start, end, Handlers...)
}

func (s *Server) Use(Handlers ...ziface.RouterHandler) ziface.IRouterSlices {
	if !s.RouterSlicesMode {
		panic("Server RouterSlicesMode is FALSE!! ")
	}
	return s.msgHandler.Use(Handlers...)
}

// 获取当前server的连接管理器
func (s *Server) GetConnMgr() ziface.IConnManager {
	return s.ConnMgr
}

// 初始化Server模块的方法
func NewServer(name string) ziface.IServer {
	s := &Server{
		Name:       zconf.GlobalObject.Name,
		IPVersion:  "tcp4",
		IP:         zconf.GlobalObject.Host,
		Port:       zconf.GlobalObject.TCPPort,
		msgHandler: NewMsgHandle(),
		ConnMgr:    NewConnManager(),
	}
	return s
}

// 注册OnConnStart hook函数方法
func (s *Server) SetOnConnStart(hookFunc func(connection ziface.IConnection)) {
	s.onConnStart = hookFunc
}

// 注册OnConnStop hook函数方法
func (s *Server) SetOnConnStop(hookFunc func(connection ziface.IConnection)) {
	s.onConnStop = hookFunc
}

// 得到该Server的连接创建时Hook函数
func (s *Server) GetOnConnStart() func(ziface.IConnection) {
	return s.onConnStart
}

// 得到该Server的连接断开时的Hook函数
func (s *Server) GetOnConnStop() func(ziface.IConnection) {
	return s.onConnStop
}

func (s *Server) GetPacket() ziface.IDataPack {
	return s.packet
}

func (s *Server) SetPacket(packet ziface.IDataPack) {
	s.packet = packet
}

func (s *Server) GetMsgHandler() ziface.IMsgHandle {
	return s.msgHandler
}

func (s *Server) GetHeartBeat() ziface.IHeartbeatChecker {
	return s.hc
}

// 启动心跳检测
func (s *Server) StartHeartBeat(interval time.Duration) {
	checker := NewHeartbeatChecker(interval)

	//添加心跳检测的路由
	s.AddRouter(checker.MsgID(), checker.Router())

	//server绑定心跳检测器
	s.hc = checker
}

// 启动心跳检测(自定义回调)
func (s *Server) StartHeartBeatWithOption(interval time.Duration, option *ziface.HeartBeatOption) {
	checker := NewHeartbeatChecker(interval)

	if option != nil {
		checker.SetHeartbeatMsgFunc(option.MakeMsg)
		checker.SetOnRemoteNotAlive(option.OnRemoteNotAlive)
		checker.BindRouter(option.HeadBeatMsgID, option.Router)
	}

	//添加心跳检测的路由
	s.AddRouter(checker.MsgID(), checker.Router())

	//server绑定心跳检测器
	s.hc = checker
}

func (s *Server) SetWebsocketAuth(f func(r *http.Request) error) {
	s.websocketAuth = f
}

func (s *Server) ServerName() string {
	return s.Name
}
