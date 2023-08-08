package znet

import "zinx_server/zinx/ziface"

type Request struct {
	//已经和客户端建立好的链接
	conn ziface.IConnection
	//客户端请求的数据
	msg      ziface.IMessage
	handlers []ziface.RouterHandler // 路由函数切片
	index    int8                   // 路由函数切片索引
}

// 得到当前链接
func (r *Request) GetConnection() ziface.IConnection {
	return r.conn
}

// 得到请求的消息数据
func (r *Request) GetData() []byte {
	return r.msg.GetData()
}

// 得到请求的消息ID
func (r *Request) GetMsgId() uint32 {
	return r.msg.GetMsgID()
}

// func (r *Request) GetMsgLen() uint32 {
// 	return r.msg.GetDataLen()
// }

// 路由切片集合 操作
func (r *Request) BindRouterSlices(handlers []ziface.RouterHandler) {
	r.handlers = handlers
}

// 路由切片操作 执行下一个函数
func (r *Request) RouterSlicesNext() {
	r.index++
	for r.index < int8(len(r.handlers)) {
		r.handlers[r.index](r)
		r.index++
	}
}
