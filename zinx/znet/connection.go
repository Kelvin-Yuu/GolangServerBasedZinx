package znet

import (
	"fmt"
	"net"
	"zinx_server/zinx/utils"
	"zinx_server/zinx/ziface"
)

/*
链接模块
*/
type Connection struct {
	//当前链接的socket TCP套接字
	Conn *net.TCPConn

	//链接的ID
	ConnID uint32

	//当前的链接状态
	isClosed bool

	// //当前链接所绑定的处理业务方式API
	// handleAPI ziface.HandleFunc

	//告知当前链接已经退出/停止的channel
	ExitChan chan bool

	//该链接处理的方法Router
	Router ziface.IRouter
}

// 初始化链接模块的方法
func NewConnection(conn *net.TCPConn, connID uint32, router ziface.IRouter) *Connection {
	c := &Connection{
		Conn:   conn,
		ConnID: connID,
		// handleAPI: callback_api,
		isClosed: false,
		ExitChan: make(chan bool, 1),
		Router:   router,
	}
	return c
}

// 链接的读业务方法
func (c *Connection) StartReader() {
	fmt.Println("Reader Goroutine is running...")
	defer fmt.Println("connID = ", c.ConnID, " Reader is exit, remote addr is ", c.RemoteAddr().String())
	defer c.Stop()

	for {
		//读取客户端的数据到buf中
		buf := make([]byte, utils.GlobalObject.MaxPackageSize)
		_, err := c.Conn.Read(buf)
		if err != nil {
			fmt.Println("recv buf err", err)
			continue
		}

		// //调用当前链接所绑定的HandleAPI
		// if err := c.handleAPI(c.Conn, buf, cnt); err != nil {
		// 	fmt.Println("ConnID ", c.ConnID, " handle is error: ", err)
		// 	continue
		// }

		//得到当前conn数据的Request请求数据
		req := Request{
			conn: c,
			data: buf,
		}

		go func(request ziface.IRequest) {
			//从路由中，找到注册绑定的Conn对应的Router调用
			c.Router.PreHandle(request)
			c.Router.Handle(request)
			c.Router.PostHandle(request)
		}(&req)

	}
}

// 启动链接 让当前的链接准备开始工作
func (c *Connection) Start() {
	fmt.Println("Conn Start()...ConnID = ", c.ConnID)

	//启动从当前链接的读数据的业务
	go c.StartReader()
	//TODO 启动从当前连接写数据业务
}

// 停止链接 结束当前链接的工作
func (c *Connection) Stop() {
	fmt.Println("Conn Stop()...ConnID = ", c.ConnID)

	//如果当前链接已经关闭
	if c.isClosed {
		return
	}
	c.isClosed = true

	//关闭socket链接
	c.Conn.Close()

	//回收资源
	close(c.ExitChan)
}

// 获取当前链接的绑定socket conn
func (c *Connection) GetTCPConnection() *net.TCPConn {
	return c.Conn
}

// 获取当前链接模块的链接ID
func (c *Connection) GetConnID() uint32 {
	return c.ConnID
}

// 获取远程客户端的TCP状态（IP,Port)
func (c *Connection) RemoteAddr() net.Addr {
	return c.Conn.RemoteAddr()
}

// 发送数据，将数据发送给远程的客户端
func (c *Connection) Send(data []byte) error {
	return nil
}