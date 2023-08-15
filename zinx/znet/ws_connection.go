package znet

import (
	"context"
	"encoding/hex"
	"errors"
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
//Websocket连接模块, 用于处理 Websocket 连接的读写业务 一个连接对应一个Connection
type WsConnection struct {
	//当前链接的socket websocket套接字
	conn *websocket.Conn
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
func newWebsocketConn(server ziface.IServer, conn *websocket.Conn, connID uint64) ziface.IConnection {
	c := &WsConnection{
		conn:        conn,
		connID:      connID,
		connIdStr:   strconv.FormatUint(connID, 10),
		isClosed:    false,
		msgBuffChan: nil,
		property:    nil,
		name:        server.ServerName(),
		localAddr:   conn.LocalAddr().String(),
		remoteAddr:  conn.RemoteAddr().String(),
	}

	lengthField := server.GetLengthField()
	if lengthField != nil {
		c.frameDecoder = zinterceptor.NewFrameDecoder(*lengthField)
	}

	c.packet = server.GetPacket()
	c.onConnStart = server.GetOnConnStart()
	c.onConnStop = server.GetOnConnStop()
	c.msgHandler = server.GetMsgHandler()

	c.connManager = server.GetConnMgr()

	server.GetConnMgr().Add(c)
	return c
}

// 创建一个Client服务端特性的连接的方法
func newWsClientConn(client ziface.IClient, conn *websocket.Conn) ziface.IConnection {
	c := &WsConnection{
		conn:        conn,
		connID:      0,  // client ignore
		connIdStr:   "", // client ignore
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

	// Inherit properties from client (从client继承过来的属性)
	c.packet = client.GetPacket()
	c.onConnStart = client.GetOnConnStart()
	c.onConnStop = client.GetOnConnStop()
	c.msgHandler = client.GetMsgHandler()

	return c
}

// 写消息Goroutine 将数据发送给客户端
func (c *WsConnection) StartWriter() {
	zlog.Ins().InfoF("Writer Goroutine is running")
	defer zlog.Ins().InfoF("%s [conn Writer exit!]", c.RemoteAddr().String())

	for {
		select {
		case data, ok := <-c.msgBuffChan:
			if ok {
				if err := c.conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
					zlog.Ins().ErrorF("Send Buff Data error:, %s Conn Writer exit", err)
					break
				}
			} else {
				zlog.Ins().ErrorF("msgBuffChan is Closed")
				break
			}
		case <-c.ctx.Done():
			return
		}
	}
}

// 读消息Goroutine，从客户端中读取数据
func (c *WsConnection) StartReader() {
	zlog.Ins().InfoF("[Reader Goroutine is running]")
	defer zlog.Ins().InfoF("%s [conn Reader exit!]", c.RemoteAddr().String())
	defer c.Stop()
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			// 从conn的IO中读取数据到内存缓存buffer中
			messageType, buffer, err := c.conn.ReadMessage()
			if err != nil {
				c.cancel()
				return
			}
			if messageType == websocket.PingMessage {
				c.updateActivity()
				continue
			}
			n := len(buffer)
			if err != nil {
				zlog.Ins().ErrorF("read msg head [read datalen=%d], error = %s", n, err.Error())
				return
			}
			zlog.Ins().DebugF("read buffer %s \n", hex.EncodeToString(buffer[0:n]))

			//正常读取到对端数据，更新心跳检测Active状态
			if n > 0 && c.hc != nil {
				c.updateActivity()
			}

			//处理自定义协议断粘包问题
			if c.frameDecoder != nil {
				// 为读取到的0-n个字节的数据进行解码
				bufArrays := c.frameDecoder.Decode(buffer)
				if bufArrays == nil {
					continue
				}
				for _, bytes := range bufArrays {
					zlog.Ins().DebugF("read buffer %s \n", hex.EncodeToString(bytes))
					msg := zpack.NewMessage(uint32(len(bytes)), bytes)
					//得到当前客户端请求的Request
					req := NewRequest(c, msg)
					c.msgHandler.Execute(req)
				}
			} else {
				msg := zpack.NewMsgPackage(uint32(n), buffer[0:n])
				req := NewRequest(c, msg)
				c.msgHandler.Execute(req)
			}
		}
	}
}

func (c *WsConnection) Start() {
	c.ctx, c.cancel = context.WithCancel(context.Background())

	c.callOnConnStart()

	//启动心跳检测
	if c.hc != nil {
		c.hc.Start()
		c.updateActivity()
	}

	// 占用workerID
	c.workerID = useWorker(c)

	go c.StartReader()

	select {
	case <-c.ctx.Done():
		c.finalizer()

		//归还workerID
		freeWorker(c)
		return
	}
}

func (c *WsConnection) Stop() {
	c.cancel()
}

func (c *WsConnection) Send(data []byte) error {
	c.msgLock.RLock()
	defer c.msgLock.RUnlock()
	if c.isClosed == true {
		return errors.New("WsConnection closed when send msg")
	}

	err := c.conn.WriteMessage(websocket.BinaryMessage, data)
	if err != nil {
		zlog.Ins().ErrorF("SendMsg err data = %+v, err = %+v", data, err)
		return err
	}
	return nil
}

func (c *WsConnection) SendToQueue(data []byte) error {
	c.msgLock.RLock()
	defer c.msgLock.RUnlock()

	if c.msgBuffChan == nil {
		c.msgBuffChan = make(chan []byte, zconf.GlobalObject.MaxMsgChanLen)

		go c.StartWriter()
	}
	idleTimeout := time.NewTimer(5 * time.Millisecond)
	defer idleTimeout.Stop()

	if c.isClosed == true {
		return errors.New("WsConnection closed when send buff msg")
	}

	if data == nil {
		zlog.Ins().ErrorF("Pack data is nil")
		return errors.New("Pack data is nil ")
	}

	select {
	case <-idleTimeout.C:
		return errors.New("send buff msg timeout")
	case c.msgBuffChan <- data:
		return nil
	}
}

// 直接将Message数据发送数据给远程的TCP客户端
func (c *WsConnection) SendMsg(msgID uint32, data []byte) error {
	c.msgLock.RLock()
	defer c.msgLock.RUnlock()

	if c.isClosed == true {
		return errors.New("WsConnection closed when send buff msg")
	}

	// 将data封包，并且发送
	msg, err := c.packet.Pack(zpack.NewMsgPackage(msgID, data))
	if err != nil {
		zlog.Ins().ErrorF("Pack error msg ID = %d", msgID)
		return errors.New("Pack error msg ")
	}
	err = c.conn.WriteMessage(websocket.BinaryMessage, msg)
	if err != nil {
		zlog.Ins().ErrorF("SendMsg err msg ID = %d, data = %+v, err = %+v", msgID, string(msg), err)
		return err
	}

	return nil
}

func (c *WsConnection) SendBuffMsg(msgID uint32, data []byte) error {
	c.msgLock.RLock()
	defer c.msgLock.RUnlock()

	if c.msgBuffChan == nil {
		c.msgBuffChan = make(chan []byte, zconf.GlobalObject.MaxMsgChanLen)
		// Start the Goroutine for writing back to the client data stream
		// This method only reads data from MsgBuffChan, allocating memory and starting Goroutine without calling SendBuffMsg
		// (开启用于写回客户端数据流程的Goroutine
		// 此方法只读取MsgBuffChan中的数据没调用SendBuffMsg可以分配内存和启用协程)
		go c.StartWriter()
	}

	idleTimeout := time.NewTimer(5 * time.Millisecond)
	defer idleTimeout.Stop()

	if c.isClosed == true {
		return errors.New("WsConnection closed when send buff msg")
	}

	// Package data and send
	// (将data封包，并且发送)
	msg, err := c.packet.Pack(zpack.NewMsgPackage(msgID, data))
	if err != nil {
		zlog.Ins().ErrorF("Pack error msg ID = %d", msgID)
		return errors.New("Pack error msg ")
	}

	// Send timeout
	select {
	case <-idleTimeout.C:
		return errors.New("send buff msg timeout")
	case c.msgBuffChan <- msg:
		return nil
	}
}

func (c *WsConnection) finalizer() {
	c.callOnConnStop()

	c.msgLock.Lock()
	c.msgLock.Unlock()

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

func (c *WsConnection) callOnConnStart() {
	if c.onConnStart != nil {
		zlog.Ins().InfoF("ZINX CallOnConnStart....")
		c.onConnStart(c)
	}
}

func (c *WsConnection) callOnConnStop() {
	if c.onConnStop != nil {
		zlog.Ins().InfoF("ZINX CallOnConnStop....")
		c.onConnStop(c)
	}
}

func (c *WsConnection) IsAlive() bool {
	if c.isClosed {
		return false
	}
	// Check the time duration since the last activity of the connection, if it exceeds the maximum heartbeat interval,
	// then the connection is considered dead
	// (检查连接最后一次活动时间，如果超过心跳间隔，则认为连接已经死亡)
	return time.Now().Sub(c.lastActivityTime) < zconf.GlobalObject.HeartbeatMaxDuration()
}

func (c *WsConnection) updateActivity() {
	c.lastActivityTime = time.Now()
}

func (c *WsConnection) GetConnection() net.Conn {
	return nil
}

func (c *WsConnection) GetWsConn() *websocket.Conn {
	return c.conn
}

// Deprecated: use GetConnection instead
func (c *WsConnection) GetTCPConnection() net.Conn {
	return nil
}

func (c *WsConnection) GetConnID() uint64 {
	return c.connID
}

func (c *WsConnection) GetConnIdStr() string {
	return c.connIdStr
}

func (c *WsConnection) GetWorkerID() uint32 {
	return c.workerID
}

func (c *WsConnection) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *WsConnection) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *WsConnection) SetProperty(key string, value interface{}) {
	c.propertyLock.Lock()
	defer c.propertyLock.Unlock()
	if c.property == nil {
		c.property = make(map[string]interface{})
	}

	c.property[key] = value
}

func (c *WsConnection) GetProperty(key string) (interface{}, error) {
	c.propertyLock.Lock()
	defer c.propertyLock.Unlock()

	if value, ok := c.property[key]; ok {
		return value, nil
	}

	return nil, errors.New("no property found")
}

func (c *WsConnection) RemoveProperty(key string) {
	c.propertyLock.Lock()
	defer c.propertyLock.Unlock()

	delete(c.property, key)
}

// 返回ctx，用于用户自定义的go程获取连接退出状态
func (c *WsConnection) Context() context.Context {
	return c.ctx
}

func (c *WsConnection) SetHeartBeat(checker ziface.IHeartbeatChecker) {
	c.hc = checker
}

func (c *WsConnection) LocalAddrString() string {
	return c.localAddr
}

func (c *WsConnection) RemoteAddrString() string {
	return c.remoteAddr
}

func (c *WsConnection) GetName() string {
	return c.name
}

func (c *WsConnection) GetMsgHandler() ziface.IMsgHandle {
	return c.msgHandler
}
