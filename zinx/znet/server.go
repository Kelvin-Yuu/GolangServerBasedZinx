package znet

import (
	"crypto/rand"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"
	"zinx_server/zinx/zconf"
	"zinx_server/zinx/zdecoder"
	"zinx_server/zinx/ziface"
	"zinx_server/zinx/zlog"
	"zinx_server/zinx/zpack"
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

	// Decoder for dealing with message fragmentation and reassembly
	// (断粘包解码器)
	decoder ziface.IDecoder

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
	fmt.Println("[conn Handle] CallBackToClient...")
	if _, err := conn.Write(data[:cnt]); err != nil {
		fmt.Println("writ back buf err: ", err)
		return errors.New("CallBackToClient error")
	}
	return nil
}

// (根据config创建一个服务器句柄)
func newServerWithConfig(config *zconf.Config, ipVersion string, opts ...Option) ziface.IServer {
	s := &Server{
		Name:             config.Name,
		IPVersion:        ipVersion,
		IP:               config.Host,
		Port:             config.TCPPort,
		WsPort:           config.WsPort,
		KcpPort:          config.KcpPort,
		msgHandler:       newMsgHandle(),
		RouterSlicesMode: config.RouterSlicesMode,
		ConnMgr:          NewConnManager(),
		exitChan:         nil,

		packet:  zpack.Factory().NewPack(ziface.ZinxDataPack),
		decoder: zdecoder.NewTLVDecoder(), // Default to using TLV decode (默认使用TLV的解码方式)
		upgrader: &websocket.Upgrader{
			ReadBufferSize: int(config.IOReadBuffSize),
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}

	for _, opt := range opts {
		opt(s)
	}
	config.Show()

	return s

}

// 创建一个服务器句柄
func NewServer(opts ...Option) ziface.IServer {
	return newServerWithConfig(zconf.GlobalObject, "tcp", opts...)
}

// 使用用户配置来创建一个服务器句柄
func NewUserConfServer(config *zconf.Config, opts ...Option) ziface.IServer {
	// 刷新用户配置到全局配置变量
	zconf.UserConfToGlobal(config)
	s := newServerWithConfig(config, "tcp4", opts...)
	return s
}

// 创建一个默认自带一个Recover处理器的服务器句柄
func NewDefaultRouterSlicesServer(opts ...Option) ziface.IServer {
	zconf.GlobalObject.RouterSlicesMode = true
	s := newServerWithConfig(zconf.GlobalObject, "tcp", opts...)
	s.Use(RouterRecovery)
	return s
}

// 创建一个用户配置的自带一个Recover处理器的服务器句柄，如果用户不希望Use这个方法，那么应该使用NewUserConfServer
func NewUserConfDefaultRouterSlicesServer(config *zconf.Config, opts ...Option) ziface.IServer {

	if !config.RouterSlicesMode {
		panic("RouterSlicesMode is false")
	}

	// 刷新用户配置到全局配置变量
	zconf.UserConfToGlobal(config)

	s := newServerWithConfig(zconf.GlobalObject, "tcp4", opts...)
	s.Use(RouterRecovery)
	return s
}

func (s *Server) StartConn(conn ziface.IConnection) {
	// HeartBeat check
	if s.hc != nil {
		// Clone一个心跳检测
		heartbeatChecker := s.hc.Clone()

		// 绑定conn
		heartbeatChecker.BindConn(conn)
	}
	// 启动conn业务
	conn.Start()
}

func (s *Server) ListenTcpConn() {
	// 1. 获取TCP地址
	addr, err := net.ResolveTCPAddr(s.IPVersion, fmt.Sprintf("%s:%d", s.IP, s.Port))
	if err != nil {
		zlog.Ins().ErrorF("[Start] resolve tcp addr err: %v\n", err)
		return
	}

	// 2. 监听TCP
	var listener net.Listener
	if zconf.GlobalObject.CertFile != "" && zconf.GlobalObject.PrivateKeyFile != "" {
		// 读取cerf和key (SSL)
		crt, err := tls.LoadX509KeyPair(zconf.GlobalObject.CertFile, zconf.GlobalObject.PrivateKeyFile)
		if err != nil {
			panic(err)
		}

		// TLS connection
		tlsConfig := &tls.Config{}
		tlsConfig.Certificates = []tls.Certificate{crt}
		tlsConfig.Time = time.Now
		tlsConfig.Rand = rand.Reader
		listener, err = tls.Listen(s.IPVersion, fmt.Sprintf("%s:%d", s.IP, s.Port), tlsConfig)
		if err != nil {
			panic(err)
		}
	} else {
		listener, err = net.ListenTCP(s.IPVersion, addr)
		if err != nil {
			panic(err)
		}
	}

	// 3. 启动服务端业务
	go func() {
		for {
			// 3.1 设置服务器最大连接控制，如果超过最大连接，则等待
			// TODO 高并发限流策略
			if s.ConnMgr.Len() >= zconf.GlobalObject.MaxConn {
				zlog.Ins().InfoF("Exceeded the maxConnNum:%d, Wait:%d", zconf.GlobalObject.MaxConn, AcceptDelay.duration)
				AcceptDelay.Delay()
				continue
			}
			// 3.2 阻塞等待客户端建立连接请求
			conn, err := listener.Accept()
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					zlog.Ins().ErrorF("Listener closed")
					return
				}
				zlog.Ins().ErrorF("Accept err: %v", err)
				AcceptDelay.Delay()
				continue
			}

			AcceptDelay.Reset()

			// 处理该新连接请求的 业务 方法， 此时应该有 handler 和 conn是绑定的
			newCid := atomic.AddUint64(&s.cID, 1)
			dealConn := newServerConn(s, conn, newCid)

			go s.StartConn(dealConn)
		}
	}()
	select {
	case <-s.exitChan:
		err := listener.Close()
		if err != nil {
			zlog.Ins().ErrorF("listener close err: %v", err)
		}
	}
}

func (s *Server) ListenWebsocketConn() {

}

func (s *Server) ListenKcpConn() {

}

// 启动服务器
func (s *Server) Start() {
	zlog.Ins().InfoF("[Zinx] Serve Name : %s, Serve Listener at IP: %s, Port: %d\n",
		s.Name, s.IP, s.Port)
	zlog.Ins().InfoF("[Zinx] Version : %s, MaxConn: %d, MaxPackageSize: %d\n",
		zconf.GlobalObject.Version,
		zconf.GlobalObject.MaxConn,
		zconf.GlobalObject.MaxPacketSize)

	// 将解码器添加到拦截器
	if s.decoder != nil {
		s.msgHandler.AddInterceptor(s.decoder)
	}
	// 启动worker工作池
	s.msgHandler.StartWorkerPool()

	// 开启一个goroutine去做服务端listener业务
	switch zconf.GlobalObject.Mode {
	case zconf.ServerModeTcp:
		go s.ListenTcpConn()
	case zconf.ServerModeWebsocket:
		go s.ListenWebsocketConn()
	case zconf.ServerModeKcp:
		go s.ListenKcpConn()
	default:
		go s.ListenTcpConn()
		go s.ListenWebsocketConn()

	}
}

// 停止服务器
func (s *Server) Stop() {
	//TODO 将一些服务器的资源、状态或者一些已经开辟的链接信息，进行停止或者回收
	zlog.Ins().InfoF("[STOP] Zinx server name %s", s.Name)
	s.ConnMgr.ClearConn()
	s.exitChan <- struct{}{}
	close(s.exitChan)
}

// 运行服务器
func (s *Server) Serve() {
	//启动server的服务功能
	s.Start()

	// 阻塞，否则主Go退出，listenner的go将会退出
	c := make(chan os.Signal, 1)
	// 监听指定信号 ctrl+c kill信号
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	sig := <-c
	zlog.Ins().InfoF("[SERVE] Zinx server , name %s, Serve Interrupt, signal = %v", s.Name, sig)
}

// 路由功能：给当前的服务注册一个路由方法，供客户端的链接处理使用
func (s *Server) AddRouter(msgID uint32, router ziface.IRouter) {
	if s.RouterSlicesMode {
		panic("Serve RouterSlicesMode is TRUE!! ")
	}
	s.msgHandler.AddRouter(msgID, router)
	fmt.Println("Add Router [MsgID =", msgID, "] Successed!")
}

func (s *Server) AddRouterSlices(msgId uint32, router ...ziface.RouterHandler) ziface.IRouterSlices {
	if !s.RouterSlicesMode {
		panic("Serve RouterSlicesMode is FALSE!! ")
	}
	return s.msgHandler.AddRouterSlices(msgId, router...)
}

func (s *Server) Group(start, end uint32, Handlers ...ziface.RouterHandler) ziface.IGroupRouterSlices {
	if !s.RouterSlicesMode {
		panic("Serve RouterSlicesMode is FALSE!! ")
	}
	return s.msgHandler.Group(start, end, Handlers...)
}

func (s *Server) Use(Handlers ...ziface.RouterHandler) ziface.IRouterSlices {
	if !s.RouterSlicesMode {
		panic("Serve RouterSlicesMode is FALSE!! ")
	}
	return s.msgHandler.Use(Handlers...)
}

// 获取当前server的连接管理器
func (s *Server) GetConnMgr() ziface.IConnManager {
	return s.ConnMgr
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
		//检测当前路由模式
		if s.RouterSlicesMode {
			checker.BindRouterSlices(option.HeartBeatMsgID, option.RouterSlices...)
		} else {
			checker.BindRouter(option.HeartBeatMsgID, option.Router)
		}
	}

	//添加心跳检测的路由
	if s.RouterSlicesMode {
		s.AddRouterSlices(checker.MsgID(), checker.RouterSlices()...)
	} else {
		s.AddRouter(checker.MsgID(), checker.Router())
	}

	//server绑定心跳检测器
	s.hc = checker
}

func (s *Server) GetLengthField() *ziface.LengthField {
	if s.decoder != nil {
		return s.decoder.GetLengthField()
	}
	return nil
}

func (s *Server) SetDecoder(decoder ziface.IDecoder) {
	s.decoder = decoder
}

func (s *Server) AddInterceptor(interceptor ziface.IInterceptor) {
	s.msgHandler.AddInterceptor(interceptor)
}

func (s *Server) SetWebsocketAuth(f func(r *http.Request) error) {
	s.websocketAuth = f
}

func (s *Server) ServerName() string {
	return s.Name
}

func init() {}
