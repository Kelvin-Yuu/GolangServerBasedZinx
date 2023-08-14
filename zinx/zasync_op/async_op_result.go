package zasync_op

import (
	"sync/atomic"
	"zinx_server/zinx/ziface"
	"zinx_server/zinx/znet"
)

type AsyncOpResult struct {
	//玩家链接
	conn ziface.IConnection
	//已返回对象
	returnObj interface{}
	//完成回调函数
	completeFunc func()
	//是否已有返回值
	hasReturnedObj int32
	//是否已有回调函数
	hasCompleteFunc int32
	//是否已经调用过完成函数（默认值=0，没被调用过）
	comleteFuncHasAlreadyBeenCalled int32
}

// 获取返回值
func (aor *AsyncOpResult) GetReturnedObj() interface{} {
	return aor.returnObj
}

// 设置返回值
func (aor *AsyncOpResult) SetReturnedObj(val interface{}) {
	if atomic.CompareAndSwapInt32(&aor.hasReturnedObj, 0, 1) {
		aor.returnObj = val
		//防止未调用回调函数的问题：设置处理结果时，直接调用回调函数：
		//1. 回调函数未绑定，调用空；
		//2. 回调函数已绑定，立马调用。
		aor.doComplete()
	}
}

// 完成回调函数
func (aor *AsyncOpResult) OnComplete(val func()) {
	if atomic.CompareAndSwapInt32(&aor.hasCompleteFunc, 0, 1) {
		aor.completeFunc = val
		//防止未调用回调函数问题:设置回调函数时，发现已经有处理结果了，直接调用
		if aor.hasReturnedObj == 1 {
			aor.doComplete()
		}
	}
}

// 执行完成回调，在设置处理结果和回调函数的时候双重检查，杜绝未调用回调函数的可能性
func (aor *AsyncOpResult) doComplete() {
	if aor.completeFunc == nil {
		return
	}
	// 防止可重入问题
	if atomic.CompareAndSwapInt32(&aor.comleteFuncHasAlreadyBeenCalled, 0, 1) {
		//防止跨线程调用问题，扔到所属业务线程里去执行
		request := znet.NewFuncRequest(aor.conn, aor.completeFunc)
		aor.conn.GetMsgHandler().SendMsgToTaskQueue(request)
	}
}

// 新建异步结果
func NewAsyncOpResult(conn ziface.IConnection) *AsyncOpResult {
	result := &AsyncOpResult{}
	result.conn = conn
	return result
}
