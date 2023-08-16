package znet

import (
	"crypto/tls"
	"fmt"
	"github.com/gorilla/websocket"
	"net"
	"time"
	"zinx_server/zinx/zconf"
	"zinx_server/zinx/zdecoder"
	"zinx_server/zinx/ziface"
	"zinx_server/zinx/zlog"
	"zinx_server/zinx/zpack"
)

type Client struct {
	//客户端名称
	Name string
	//目标链接服务器的IP
	Ip string
	//目标链接服务器的端口
	Port int
	//客户端版本
	version string
	//链接实例
	conn ziface.IConnection
	//该client的连接创建时hook函数
	onConnStart func(conn ziface.IConnection)
	//该client的连接断开时hook函数
	onConnStop func(conn ziface.IConnection)
	//数据报文的封包方式
	packet ziface.IDataPack
	//异步获取连接关闭状态
	exitChan chan struct{}
	//消息管理模块
	msgHandler ziface.IMsgHandle
	//断粘包解码器
	decoder ziface.IDecoder
	//心跳检测器
	hc ziface.IHeartbeatChecker
	//使用TLS
	useTLS bool
	//websocket 链接
	dialer *websocket.Dialer
	//Error Channel
	ErrChan chan error
}

func NewClient(ip string, port int, opts ...ClientOption) ziface.IClient {
	c := &Client{
		Name:    "ZinxClientTcp",
		Ip:      ip,
		Port:    port,
		version: "tcp",

		msgHandler: newMsgHandle(),
		packet:     zpack.Factory().NewPack(ziface.ZinxDataPack),
		decoder:    zdecoder.NewTLVDecoder(),
		ErrChan:    make(chan error),
	}
	//应用Option设置
	for _, opt := range opts {
		opt(c)
	}

	return c
}

func NewWsClient(ip string, port int, opts ...ClientOption) ziface.IClient {

	c := &Client{
		// Default name, can be modified using the WithNameClient Option
		// (默认名称，可以使用WithNameClient的Option修改)
		Name: "ZinxClientWs",
		Ip:   ip,
		Port: port,

		msgHandler: newMsgHandle(),
		packet:     zpack.Factory().NewPack(ziface.ZinxDataPack), // Default to using Zinx's TLV packet format(默认使用zinx的TLV封包方式)
		decoder:    zdecoder.NewTLVDecoder(),                     // Default to using Zinx's TLV decoder(默认使用zinx的TLV解码器)
		version:    "websocket",
		dialer:     &websocket.Dialer{},
		ErrChan:    make(chan error),
	}

	// Apply Option settings (应用Option设置)
	for _, opt := range opts {
		opt(c)
	}

	return c
}

func NewTLSClient(ip string, port int, opts ...ClientOption) ziface.IClient {

	c, _ := NewClient(ip, port, opts...).(*Client)

	c.useTLS = true

	return c
}

// 重启客户端
func (c *Client) Restart() {
	c.exitChan = make(chan struct{})
	//客户端将协程池关闭
	zconf.GlobalObject.WorkerPoolSize = 0

	go func() {
		addr := &net.TCPAddr{
			IP:   net.ParseIP(c.Ip),
			Port: c.Port,
			Zone: "",
		}

		//创建原始Socket，得到net.Conn
		switch c.version {
		case "websocket":
			wsAddr := fmt.Sprintf("ws://%s:%d", c.Ip, c.Port)

			wsConn, _, err := c.dialer.Dial(wsAddr, nil)
			if err != nil {
				zlog.Ins().ErrorF("WsClient connect to server failed, err:%v", err)
				c.ErrChan <- err
				return
			}
			c.conn = newWsClientConn(c, wsConn)
		default:
			var conn net.Conn
			var err error
			if c.useTLS {
				// TLS encryption
				config := &tls.Config{
					// Skip certificate verification here because the CA certificate of the certificate issuer is not authenticated
					// (这里是跳过证书验证，因为证书签发机构的CA证书是不被认证的)
					InsecureSkipVerify: true,
				}

				conn, err = tls.Dial("tcp", fmt.Sprintf("%v:%v", net.ParseIP(c.Ip), c.Port), config)
				if err != nil {
					zlog.Ins().ErrorF("tls client connect to server failed, err:%v", err)
					c.ErrChan <- err
					return
				}
			} else {
				conn, err = net.DialTCP("tcp", nil, addr)
				if err != nil {
					// connection failed
					zlog.Ins().ErrorF("client connect to server failed, err:%v", err)
					c.ErrChan <- err
					return
				}
			}
			// Create Connection object
			c.conn = newClientConn(c, conn)
		}

		zlog.Ins().InfoF("[START] Zinx Client LocalAddr: %s, RemoteAddr: %s\n", c.conn.LocalAddr(), c.conn.RemoteAddr())
		// HeartBeat detection
		if c.hc != nil {
			// Bind connection and heartbeat detector after connection is successfully established
			// (创建链接成功，绑定链接与心跳检测器)
			c.hc.BindConn(c.conn)
		}

		// Start connection
		go c.conn.Start()

		select {
		case <-c.exitChan:
			zlog.Ins().InfoF("client exit.")
		}
	}()

}

// 启动客户端
func (c *Client) Start() {
	//将解码器添加到拦截器
	if c.decoder != nil {
		c.msgHandler.AddInterceptor(c.decoder)
	}
	c.Restart()
}

// 停止客户端
func (c *Client) Stop() {
	zlog.Ins().InfoF("[STOP] Zinx Client LocalAddr: %s, RemoteAddr: %s\n", c.conn.LocalAddr(), c.conn.RemoteAddr())
	c.conn.Stop()
	c.exitChan <- struct{}{}
	close(c.exitChan)
	close(c.ErrChan)
}

// 路由功能：给当前的服务注册一个路由方法，供客户端的链接处理使用
func (c *Client) AddRouter(msgID uint32, router ziface.IRouter) {
	c.msgHandler.AddRouter(msgID, router)
}

// 获取当前Client的连接
func (c *Client) Conn() ziface.IConnection {
	return c.conn
}

// 注册OnConnStart hook函数方法
func (c *Client) SetOnConnStart(hookFunc func(connection ziface.IConnection)) {
	c.onConnStart = hookFunc
}

// 注册OnConnStop hook函数方法
func (c *Client) SetOnConnStop(hookFunc func(connection ziface.IConnection)) {
	c.onConnStop = hookFunc
}

// 获取该Client的连接创建时Hook函数
func (c *Client) GetOnConnStart() func(ziface.IConnection) {
	return c.onConnStart
}

// 设置该Client的连接断开时的Hook函数
func (c *Client) GetOnConnStop() func(ziface.IConnection) {
	return c.onConnStop
}

// 获取Client绑定的数据协议封包方法
func (c *Client) GetPacket() ziface.IDataPack {
	return c.packet
}

// 设置Client绑定的数据协议封包方法
func (c *Client) SetPacket(pack ziface.IDataPack) {
	c.packet = pack
}

// 获取Client绑定的消息处理模块
func (c *Client) GetMsgHandler() ziface.IMsgHandle {
	return c.msgHandler
}

// 启动心跳检测
func (c *Client) StartHeartBeat(interval time.Duration) {
	checker := NewHeartbeatChecker(interval)

	//添加心跳检测的路由
	c.AddRouter(checker.MsgID(), checker.Router())

	c.hc = checker
}

// 启动心跳检测（自定义回调）
func (c *Client) StartHeartBeatWithOption(interval time.Duration, option *ziface.HeartBeatOption) {
	checker := NewHeartbeatChecker(interval)

	if option != nil {
		checker.SetHeartbeatMsgFunc(option.MakeMsg)
		checker.SetOnRemoteNotAlive(option.OnRemoteNotAlive)
		checker.BindRouter(option.HeartBeatMsgID, option.Router)
	}

	c.AddRouter(checker.MsgID(), checker.Router())

	c.hc = checker
}

// 获取客户端的长度字段
func (c *Client) GetLengthField() *ziface.LengthField {
	if c.decoder != nil {
		return c.decoder.GetLengthField()
	}
	return nil
}

// 设置解码器
func (c *Client) SetDecoder(decoder ziface.IDecoder) {
	c.decoder = decoder
}

// 添加拦截器
func (c *Client) AddInterceptor(interceptor ziface.IInterceptor) {
	c.msgHandler.AddInterceptor(interceptor)
}

// 获取客户端错误管道
func (c *Client) GetErrChan() chan error {
	return c.ErrChan
}

// 设置客户端Client名称
func (c *Client) SetName(name string) {
	c.Name = name
}

// 获取客户端Client名称
func (c *Client) GetName() string {
	return c.Name
}
