package znet

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"time"
	"zinx_server/zinx/zconf"
	"zinx_server/zinx/ziface"
	"zinx_server/zinx/zlog"
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
	MsgHandler ziface.IMsgHandle

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

	//// Framedecoder for solving fragmentation and packet sticking problems
	//// (断粘包解码器)
	//frameDecoder ziface.IFrameDecoder

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

// 初始化链接模块的方法
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

	//从server中继承过来的属性
	c.packet = server.GetPacket()
	c.onConnStart = server.GetOnConnStart()
	c.onConnStop = server.GetOnConnStop()
	c.MsgHandler = server.GetMsgHandler()

	//将当前的Conn与Server的ConnManager绑定
	c.connManager = server.GetConnMgr()

	//将新创建的conn加入到ConnManager中
	server.GetConnMgr().Add(c)

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

		}

	}

	for {
		//创建一个拆包解包对象
		dp := NewDataPack()

		//读取客户端的Msg Head 二进制流 8 bytes
		headData := make([]byte, dp.GetHeadLen())
		n, err := io.ReadFull(c.GetTCPConnection(), headData)
		if err != nil {
			fmt.Println("read msg head error: ", err)
			break
		}

		// 正常读取到对端数据，更新心跳检测Active状态
		if n > 0 && c.hc != nil {
			c.updateActivity()
		}

		//拆包，得到msgID 和 msgDataLen 放在msg消息中
		msg, err := dp.UnPack(headData)
		if err != nil {
			fmt.Println("unpack error: ", err)
			break
		}

		//根据dataLen，再次读取Data，放在msg.Data中
		if msg.GetDataLen() > 0 {
			var data []byte = make([]byte, msg.GetDataLen())
			if _, err := io.ReadFull(c.GetTCPConnection(), data); err != nil {
				fmt.Println("read msg data error: ", err)
				break
			}
			msg.SetData(data)
		}

		//得到当前conn数据的Request请求数据
		req := Request{
			conn: c,
			msg:  msg,
		}

		if zconf.GlobalObject.WorkerPoolSize > 0 {
			//已经开启了工作池机制，将消息发送给我Worker工作池处理
			c.MsgHandler.SendMsgToTaskQueue(&req)
		} else {
			//从路由中，找到注册绑定的Conn对应的Router调用
			//根据绑定好的MsgID找到对应处理api业务方法
			go c.MsgHandler.DoMsgHandle(&req)
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
	fmt.Println("conn Start()...connID = ", c.connID)

	// 启动心跳检测
	if c.hc != nil {
		c.hc.Start()
		c.updateActivity()
	}

	//启动从当前链接的读数据的业务
	go c.StartReader()
	//启动从当前连接写数据业务
	go c.StartWriter()

	//按照开发者传递进来的 创建链接之后需要调用的处理业务，执行对应的hook函数
	c.callOnConnStop()
}

// 停止链接 结束当前链接的工作
func (c *Connection) Stop() {
	fmt.Println("conn Stop()...connID = ", c.connID)

	//按照开发者传递进来的 销毁链接之前需要调用的处理业务，执行对应的hook函数
	c.callOnConnStart()

	//如果当前链接已经关闭SendMsg
	if c.isClosed {
		return
	}

	// 关闭链接绑定的心跳检测器
	if c.hc != nil {
		c.hc.Stop()
	}

	//关闭socket链接
	c.conn.Close()

	//告知Writer关闭
	c.ExitChan <- true

	//将当前链接从ConnMgr中删除掉
	if c.connManager != nil {
		c.connManager.Remove(c)
	}

	//回收资源
	close(c.ExitChan)
	close(c.msgChan)

	c.isClosed = true
}

// 获取当前链接的绑定socket conn
func (c *Connection) GetTCPConnection() *net.TCPConn {
	return c.conn
}

// 获取当前链接模块的链接ID
func (c *Connection) GetConnID() uint32 {
	return c.connID
}

// 获取远程客户端的TCP状态（IP,Port)
func (c *Connection) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

// 获取本地服务器的TCP状态（IP,Port）
func (c *Connection) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

// 提供一个SendMsg方法 将我们要发送给客户端的数据，先进行封包，再发送
func (c *Connection) SendMsg(msgId uint32, data []byte) error {
	if c.isClosed {
		return errors.New("Connection closed when send msg")
	}

	//将data进行封包 MsgDataLen|MsgID|Data
	dp := NewDataPack()
	binaryMsg, err := dp.Pack(NewMsgPackage(msgId, data))
	if err != nil {
		fmt.Println("Pack error msg id = ", msgId)
		return errors.New("Pack error msg")
	}

	//发送消息到绑定客户端
	c.msgChan <- binaryMsg

	return nil
}

// 提供一个SendMsg方法 将我们要发送给客户端的数据，先进行封包，再发送(有缓冲)
func (c *Connection) SendBuffMsg(msgId uint32, data []byte) error {
	if c.isClosed {
		return errors.New("Connection closed when send buff msg")
	}
	//将data封包，并且发送
	dp := NewDataPack()
	msg, err := dp.Pack(NewMsgPackage(msgId, data))
	if err != nil {
		fmt.Println("Pack error msg id = ", msgId)
		return errors.New("Pack error msg ")
	}

	//发送消息到绑定客户端
	c.msgBuffChan <- msg

	return nil
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
		fmt.Println("----> Call OnConnStart()...")
		c.onConnStart(c)
	}
}

// 调用onConnStop Hook函数
func (c *Connection) callOnConnStop() {
	fmt.Println("----> Call OnConnStop()...")
	c.onConnStop(c)
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
