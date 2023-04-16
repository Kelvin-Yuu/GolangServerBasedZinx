package znet

import (
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
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
	//告知当前链接已经退出/停止的channel
	ExitChan chan bool
	//当前链接是属于哪个Connection Manager的
	connManager ziface.IConnManager
	//无缓冲管道，用于读写Goroutine之间的消息通信
	msgChan chan []byte
	//有缓冲管道，用于读写Goroutine之间的消息通信
	msgBuffChan chan []byte
	//消息的管理MsgID和对应的处理业务API关系
	MsgHandler ziface.IMsgHandle
	//链接属性集合
	property map[string]interface{}
	//保护链接属性的锁
	propertyLock sync.RWMutex
	// 前连接创建时Hook函数
	onConnStart func(conn ziface.IConnection)
	//当前连接断开时的Hook函数
	onConnStop func(conn ziface.IConnection)
	// 最后一次活动时间
	lastActivityTime time.Time
	// 心跳检测器
	hc ziface.IHeartbeatChecker
}

// 初始化链接模块的方法
func NewConnection(server ziface.IServer, conn *net.TCPConn, connID uint32, msgHandler ziface.IMsgHandle) *Connection {
	c := &Connection{
		Conn:        conn,
		ConnID:      connID,
		isClosed:    false,
		ExitChan:    make(chan bool, 1),
		msgChan:     make(chan []byte),
		msgBuffChan: make(chan []byte, utils.GlobalObject.MaxMsgChanLen),
		MsgHandler:  msgHandler,
		property:    make(map[string]interface{}),
	}

	//从server中继承过来的属性
	c.onConnStart = server.GetOnConnStart()
	c.onConnStop = server.GetOnConnStop()

	//将当前的Conn与Server的ConnManager绑定
	c.connManager = server.GetConnMgr()

	//将新创建的conn加入到ConnManager中
	server.GetConnMgr().Add(c)

	return c
}

// 链接的读业务方法
func (c *Connection) StartReader() {

	fmt.Println("[Reader Goroutine is running...]")
	defer fmt.Println("[Reader is exit] connID = ", c.ConnID, " remote addr is ", c.RemoteAddr().String())
	defer c.Stop()

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

		if utils.GlobalObject.WorkerPoolSize > 0 {
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
	fmt.Println("[Writer Goroutine is running...]")
	defer fmt.Println(c.RemoteAddr().String(), " [conn Writer exit!]")

	//不断阻塞等待channel消息，进行写给客户端
	for {
		select {
		case data := <-c.msgChan:
			if _, err := c.Conn.Write(data); err != nil {
				fmt.Println("Send data Error:", err)
				return
			}
		case data, ok := <-c.msgBuffChan:
			if ok {
				//有数据要写给客户端
				if _, err := c.Conn.Write(data); err != nil {
					fmt.Println("Send Buff Data Error:, ", err, " Conn Writer exit")
					return
				}
			} else {
				fmt.Println("msgBuffChan is Closed")
				break
			}
		case <-c.ExitChan:
			//代表Reader已经退出，此时Writer也要退出
			return
		}
	}
}

// 启动链接 让当前的链接准备开始工作
func (c *Connection) Start() {
	fmt.Println("Conn Start()...ConnID = ", c.ConnID)

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
	fmt.Println("Conn Stop()...ConnID = ", c.ConnID)

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
	c.Conn.Close()

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

// 获取本地服务器的TCP状态（IP,Port）
func (c *Connection) LocalAddr() net.Addr {
	return c.Conn.LocalAddr()
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
	return time.Now().Sub(c.lastActivityTime) < utils.GlobalObject.HeartbeatMaxDuration()
}

// 更新连接最后活动时间
func (c *Connection) updateActivity() {
	c.lastActivityTime = time.Now()
}

// 设置心跳检测器
func (c *Connection) SetHeartBeat(checker ziface.IHeartbeatChecker) {
	c.hc = checker
}
