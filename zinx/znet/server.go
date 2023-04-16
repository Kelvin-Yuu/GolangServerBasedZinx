package znet

import (
	"errors"
	"fmt"
	"net"
	"time"
	"zinx_server/zinx/utils"
	"zinx_server/zinx/ziface"
)

// iServer的接口实现，定义一个Server的服务器模块
type Server struct {
	//服务器的名称
	Name string

	//服务器绑定的ip版本
	IPVersion string

	//服务器监听的IP
	IP string

	//服务器监听的端口
	Port int

	//当前server的消息管理模块，用来绑定MsgID和对应的处理业务API
	MsgHandler ziface.IMsgHandle

	//当前server的连接管理模块
	ConnManager ziface.IConnManager

	//当前server创建链接之后自动调用Hook函数
	OnConnStart func(conn ziface.IConnection)

	//当前server销毁链接之前自动调用Hook函数
	OnConnStop func(conn ziface.IConnection)

	//心跳检测器
	hc ziface.IHeartbeatChecker
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
	fmt.Printf("[Zinx] Server Name : %s, Server Listener at IP: %s, Port: %d\n",
		utils.GlobalObject.Name,
		utils.GlobalObject.Host,
		utils.GlobalObject.TCPPort)
	fmt.Printf("[Zinx] Version : %s, MaxConn: %d, MaxPackageSize: %d\n",
		utils.GlobalObject.Version,
		utils.GlobalObject.MaxConn,
		utils.GlobalObject.MaxPackageSize)

	go func() {

		// 0 开启消息队列及Worker工作池
		s.MsgHandler.StartWorkerPool()

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
			if s.ConnManager.Len() >= utils.GlobalObject.MaxConn {
				//TODO 给客户端响应一个超出最大连接的错误包
				fmt.Println("Too many Connection, MaxConn=", utils.GlobalObject.MaxConn)
				conn.Close()
				continue
			}

			//已经与客户端建立连接，做某些业务
			//将处理新链接的业务方法和conn进行绑定，得到我们的链接模块
			dealConn := NewConnection(s, conn, cid, s.MsgHandler)
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
	s.ConnManager.ClearConn()
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
	s.MsgHandler.AddRouter(msgID, router)
	fmt.Println("Add Router [MsgID =", msgID, "] Successed!")
}

// 获取当前server的连接管理器
func (s *Server) GetConnMgr() ziface.IConnManager {
	return s.ConnManager
}

// 初始化Server模块的方法
func NewServer(name string) ziface.IServer {
	s := &Server{
		Name:        utils.GlobalObject.Name,
		IPVersion:   "tcp4",
		IP:          utils.GlobalObject.Host,
		Port:        utils.GlobalObject.TCPPort,
		MsgHandler:  NewMsgHandle(),
		ConnManager: NewConnManager(),
	}
	return s
}

// 注册OnConnStart hook函数方法
func (s *Server) SetOnConnStart(hookFunc func(connection ziface.IConnection)) {
	s.OnConnStart = hookFunc
}

// 注册OnConnStop hook函数方法
func (s *Server) SetOnConnStop(hookFunc func(connection ziface.IConnection)) {
	s.OnConnStop = hookFunc
}

// 得到该Server的连接创建时Hook函数
func (s *Server) GetOnConnStart() func(ziface.IConnection) {
	return s.OnConnStart
}

// 得到该Server的连接断开时的Hook函数
func (s *Server) GetOnConnStop() func(ziface.IConnection) {
	return s.OnConnStop
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
