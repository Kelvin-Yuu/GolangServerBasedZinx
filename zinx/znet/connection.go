package znet

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"net"
	"strconv"
	"sync"
	"time"
	"zinx_server/zinx/zconf"
	"zinx_server/zinx/ziface"
	"zinx_server/zinx/zinterceptor"
	"zinx_server/zinx/zlog"
	"zinx_server/zinx/zpack"
)

/*
链接模块
*/
type Connection struct {
	//当前链接的socket TCP套接字
	conn net.Conn
	//链接的ID，理论支持的进程connID的最大数量0 ~ 18,446,744,073,709,551,615
	connID uint64
	//字符串的链接ID
	connIdStr string
	//负责处理该链接的workId
	workerID uint32

	//消息的管理MsgID和对应的处理业务API关系
	msgHandler ziface.IMsgHandle

	//告知该链接已经退出/停止的ctx
	ctx    context.Context
	cancel context.CancelFunc

	//有缓冲管道，用于读写Goroutine之间的消息通信
	msgBuffChan chan []byte

	//用户收发消息的Lock
	msgLock sync.RWMutex

	//链接属性集合
	property map[string]interface{}
	//保护链接属性的锁
	propertyLock sync.RWMutex

	//当前的链接状态
	isClosed bool

	//当前链接是属于哪个Connection Manager的
	connManager ziface.IConnManager

	// 前连接创建时Hook函数
	onConnStart func(conn ziface.IConnection)
	//当前连接断开时的Hook函数
	onConnStop func(conn ziface.IConnection)

	// Data packet packaging method
	// (数据报文封包方式)
	packet ziface.IDataPack

	// Framedecoder for solving fragmentation and packet sticking problems
	// (断粘包解码器)
	frameDecoder ziface.IFrameDecoder

	// 最后一次活动时间
	lastActivityTime time.Time
	// 心跳检测器
	hc ziface.IHeartbeatChecker

	// 链接名称，默认与创建链接的Server/Client的Name一致
	name string

	// 当前链接的本地地址
	localAddr string

	// 当前链接的远程地址
	remoteAddr string
}

// 创建一个Server服务端特性的连接的方法
func newServerConn(server ziface.IServer, conn net.Conn, connID uint64) *Connection {
	c := &Connection{
		conn:        conn,
		connID:      connID,
		connIdStr:   strconv.FormatUint(connID, 10),
		isClosed:    false,
		msgBuffChan: make(chan []byte, zconf.GlobalObject.MaxMsgChanLen),
		property:    nil,
		name:        server.ServerName(),
		localAddr:   conn.LocalAddr().String(),
		remoteAddr:  conn.RemoteAddr().String(),
	}

	lengthField := server.GetLengthField()
	if lengthField != nil {
		c.frameDecoder = zinterceptor.NewFrameDecoder(*lengthField)
	}

	//从server中继承过来的属性
	c.packet = server.GetPacket()
	c.onConnStart = server.GetOnConnStart()
	c.onConnStop = server.GetOnConnStop()
	c.msgHandler = server.GetMsgHandler()

	//将当前的Conn与Server的ConnManager绑定
	c.connManager = server.GetConnMgr()

	//将新创建的conn加入到ConnManager中
	server.GetConnMgr().Add(c)

	return c
}

// 创建一个Client服务端特性的连接的方法
func newClientConn(client ziface.IClient, conn net.Conn) ziface.IConnection {
	c := &Connection{
		conn:        conn,
		connID:      0,
		connIdStr:   "",
		isClosed:    false,
		msgBuffChan: nil,
		property:    nil,
		name:        client.GetName(),
		localAddr:   conn.LocalAddr().String(),
		remoteAddr:  conn.RemoteAddr().String(),
	}

	lengthField := client.GetLengthField()
	if lengthField != nil {
		c.frameDecoder = zinterceptor.NewFrameDecoder(*lengthField)
	}

	c.packet = client.GetPacket()
	c.onConnStart = client.GetOnConnStart()
	c.onConnStop = client.GetOnConnStop()
	c.msgHandler = client.GetMsgHandler()

	return c
}

// 链接的读业务方法
func (c *Connection) StartReader() {
	zlog.Ins().InfoF("[Reader Goroutine is running]")
	defer zlog.Ins().InfoF("%s [conn Reader exit!]", c.RemoteAddr().String())
	defer c.Stop()
	defer func() {
		if err := recover(); err != nil {
			zlog.Ins().ErrorF("connID=%d, panic err=%v", c.GetConnID(), err)
		}
	}()

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			buffer := make([]byte, zconf.GlobalObject.IOReadBuffSize)

			// 从conn的IO中读取数据到内存缓存buffer中
			n, err := c.conn.Read(buffer)
			if err != nil {
				zlog.Ins().ErrorF("read msg head [read datalen=%d], error = %s", n, err)
				return
			}
			zlog.Ins().DebugF("read buffer %s \n", hex.EncodeToString(buffer[0:n]))

			// 正常读取到对端数据，更新心跳检测Active状态
			if n > 0 && c.hc != nil {
				c.updateActivity()
			}

			// 处理自定义协议断粘包问题
			if c.frameDecoder != nil {
				// 为读取到的0-n个字节的数据进行解码
				bufArrays := c.frameDecoder.Decode(buffer[0:n])
				if bufArrays == nil {
					continue
				}
				for _, bytes := range bufArrays {
					msg := zpack.NewMessage(uint32(len(bytes)), bytes)
					// 得到当前客户端请求的Request数据
					req := NewRequest(c, msg)
					c.msgHandler.Execute(req)
				}
			} else {
				msg := zpack.NewMessage(uint32(n), buffer[0:n])
				// 得到当前客户端请求的Request数据
				req := NewRequest(c, msg)
				c.msgHandler.Execute(req)
			}
		}
	}
}

// 链接的写业务方法
// 写消息的Goroutine，专门发送给Client消息的模块
func (c *Connection) StartWriter() {
	zlog.Ins().InfoF("Writer Goroutine is running")
	defer zlog.Ins().InfoF("%s [conn Writer exit!]", c.RemoteAddr().String())

	//不断阻塞等待channel消息，进行写给客户端
	for {
		select {
		case data, ok := <-c.msgBuffChan:
			if ok {
				//有数据要写给客户端
				if _, err := c.conn.Write(data); err != nil {
					zlog.Ins().ErrorF("Send Buff Data error:, %s Conn Writer exit", err)
					return
				}
			} else {
				fmt.Println("msgBuffChan is Closed")
				break
			}
		case <-c.ctx.Done():
			//代表Reader已经退出，此时Writer也要退出
			return
		}
	}
}

// 启动链接 让当前的链接准备开始工作
func (c *Connection) Start() {
	defer func() {
		if err := recover(); err != nil {
			zlog.Ins().ErrorF("Connection Start() error: %v", err)
		}
	}()
	c.ctx, c.cancel = context.WithCancel(context.Background())

	// 按照用户传递进来的 创建连接时需要处理的业务，执行hook方法
	c.callOnConnStart()

	// 启动心跳检测
	if c.hc != nil {
		c.hc.Start()
		c.updateActivity()
	}

	// 占用workerID
	c.workerID = useWorker(c)

	//启动从当前链接的读数据的业务
	go c.StartReader()

	select {
	case <-c.ctx.Done():
		c.finalizer()

		//归还workerID
		freeWorker(c)
		return
	}
}

// 停止链接 结束当前链接的工作
func (c *Connection) Stop() {
	c.cancel()
}

// 获取当前链接的绑定socket conn
func (c *Connection) GetConnection() net.Conn {
	return c.conn
}

// Deprecated: use GetConnection instead
func (c *Connection) GetTCPConnection() net.Conn {
	return c.conn
}

func (c *Connection) GetWsConn() *websocket.Conn {
	return nil
}

// 获取当前链接模块的链接ID
func (c *Connection) GetConnID() uint64 {
	return c.connID
}

func (c *Connection) GetConnIdStr() string {
	return c.connIdStr
}

func (c *Connection) GetMsgHandler() ziface.IMsgHandle {
	return c.msgHandler
}

func (c *Connection) GetWorkerID() uint32 {
	return c.workerID
}

// 获取远程客户端的TCP状态（IP,Port)
func (c *Connection) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}
func (c *Connection) RemoteAddrString() string {
	return c.remoteAddr
}

// 获取本地服务器的TCP状态（IP,Port）
func (c *Connection) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *Connection) LocalAddrString() string {
	return c.localAddr
}

func (c *Connection) Send(data []byte) error {
	c.msgLock.RLock()
	defer c.msgLock.RUnlock()

	if c.isClosed == true {
		return errors.New("connection closed when send msg")
	}

	_, err := c.conn.Write(data)
	if err != nil {
		zlog.Ins().ErrorF("SendMsg err data = %+v, err = %+v", data, err)
		return err
	}

	return nil
}

func (c *Connection) SendToQueue(data []byte) error {
	c.msgLock.RLock()
	defer c.msgLock.RUnlock()

	if c.isClosed == true {
		return errors.New("connection closed when send buff msg")
	}

	if c.msgBuffChan == nil {
		c.msgBuffChan = make(chan []byte, zconf.GlobalObject.MaxMsgChanLen)
		// 开启用于写回客户端数据流程的Goroutine
		// 此方法只读取MsgBuffChan中的数据没调用SendBuffMsg可以分配内存和启用协程
		go c.StartWriter()
	}

	idleTimeout := time.NewTimer(5 * time.Millisecond)
	defer idleTimeout.Stop()

	if data == nil {
		zlog.Ins().ErrorF("Pack data is nil")
		return errors.New("Pack data is nil")
	}

	// Send timeout
	select {
	case <-idleTimeout.C:
		return errors.New("send buff timeout")
	case c.msgBuffChan <- data:
		return nil
	}
}

// 提供一个SendMsg方法 将我们要发送给客户端的数据，先进行封包，再发送
func (c *Connection) SendMsg(msgId uint32, data []byte) error {
	if c.isClosed == true {
		return errors.New("connection closed when send msg")
	}
	// Pack data and send it
	msg, err := c.packet.Pack(zpack.NewMsgPackage(msgId, data))
	if err != nil {
		zlog.Ins().ErrorF("Pack error msg ID = %d", msgId)
		return errors.New("Pack error msg ")
	}

	err = c.Send(msg)
	if err != nil {
		zlog.Ins().ErrorF("SendMsg err msg ID = %d, data = %+v, err = %+v", msgId, string(msg), err)
		return err
	}

	return nil
}

// 提供一个SendMsg方法 将我们要发送给客户端的数据，先进行封包，再发送(有缓冲)
func (c *Connection) SendBuffMsg(msgId uint32, data []byte) error {
	if c.isClosed == true {
		return errors.New("connection closed when send buff msg")
	}
	if c.msgBuffChan == nil {
		c.msgBuffChan = make(chan []byte, zconf.GlobalObject.MaxMsgChanLen)
		// 开启用于写回客户端数据流程的Goroutine
		// 此方法只读取MsgBuffChan中的数据没调用SendBuffMsg可以分配内存和启用协程
		go c.StartWriter()
	}

	idleTimeout := time.NewTimer(5 * time.Millisecond)
	defer idleTimeout.Stop()

	msg, err := c.packet.Pack(zpack.NewMsgPackage(msgId, data))
	if err != nil {
		zlog.Ins().ErrorF("Pack error msg ID = %d", msgId)
		return errors.New("Pack error msg ")
	}

	// send timeout
	select {
	case <-idleTimeout.C:
		return errors.New("send buff msg timeout")
	case c.msgBuffChan <- msg:
		return nil
	}
}

// 设置链接属性
func (c *Connection) SetProperty(key string, value interface{}) {
	c.propertyLock.Lock()
	defer c.propertyLock.Unlock()

	//添加一个链接属性
	c.property[key] = value
}

// 获取链接属性
func (c *Connection) GetProperty(key string) (interface{}, error) {
	c.propertyLock.RLock()
	defer c.propertyLock.RUnlock()

	if value, ok := c.property[key]; ok {
		return value, nil
	} else {
		return nil, errors.New("no property found!")
	}
}

// 移除链接属性
func (c *Connection) RemoveProperty(key string) {
	c.propertyLock.Lock()
	defer c.propertyLock.Unlock()

	//删除属性
	delete(c.property, key)
}

// 调用onConnStart Hook函数
func (c *Connection) callOnConnStart() {
	if c.onConnStart != nil {
		zlog.Ins().InfoF("ZINX CallOnConnStart....")
		c.onConnStart(c)
	}
}

// 调用onConnStop Hook函数
func (c *Connection) callOnConnStop() {
	if c.onConnStop != nil {
		zlog.Ins().InfoF("ZINX CallOnConnStop....")
		c.onConnStop(c)
	}
}

// 判断当前链接是否存活
func (c *Connection) IsAlive() bool {
	if c.isClosed {
		return false
	}

	//检查连接最后一次活动时间，如果超出心跳间隔，则认为连接已经死亡
	return time.Now().Sub(c.lastActivityTime) < zconf.GlobalObject.HeartbeatMaxDuration()
}

// 更新连接最后活动时间
func (c *Connection) updateActivity() {
	c.lastActivityTime = time.Now()
}

// 设置心跳检测器
func (c *Connection) SetHeartBeat(checker ziface.IHeartbeatChecker) {
	c.hc = checker
}

func (c *Connection) finalizer() {
	// 如果用户注册了该链接， 那么这里直接调用关闭回调业务
	c.callOnConnStop()

	c.msgLock.Lock()
	defer c.msgLock.Unlock()

	if c.isClosed == true {
		return
	}

	if c.hc != nil {
		c.hc.Stop()
	}

	_ = c.conn.Close()

	if c.connManager != nil {
		c.connManager.Remove(c)
	}

	if c.msgBuffChan != nil {
		close(c.msgBuffChan)
	}

	c.isClosed = true

	zlog.Ins().InfoF("Conn Stop()...ConnID = %d", c.connID)

}
