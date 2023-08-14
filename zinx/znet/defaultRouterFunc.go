package znet

import (
	"bytes"
	"fmt"
	"path"
	"runtime"
	"strings"
	"time"
	"zinx_server/zinx/ziface"
	"zinx_server/zinx/zlog"
)

const (
	//开始追踪堆栈信息的层数
	StackBegin = 3
	//追踪到最后的层数
	StackEnd = 5
)

/*
	用来存放一些RouterSlicesMode下的路由可用的默认中间件
*/

// 接受业务执行上产生的panic并且尝试记录现场信息
func RouterRecovery(request ziface.IRequest) {
	defer func() {
		if err := recover(); err != nil {
			panicInfo := getInfo(StackBegin)
			zlog.Ins().ErrorF("MsgId:%d Handler panic: info:%s err:%v", request.GetMsgID(), panicInfo, err)
		}
	}()
	request.RouterSlicesNext()
}

// 简单累计所有路由组的耗时，不启用
func RouterTime(request ziface.IRequest) {
	now := time.Now()
	request.RouterSlicesNext()
	duration := time.Since(now)
	fmt.Println(duration.String())
}

func getInfo(ship int) (infoStr string) {
	panicInfo := new(bytes.Buffer)
	//也可以不指定终点层数即i := ship;; i++ 通过if！ok 结束循环，但是会一直追到最底层报错信息
	for i := ship; i <= StackEnd; i++ {
		pc, file, lineNo, ok := runtime.Caller(i)
		if !ok {
			break
		}
		funcName := runtime.FuncForPC(pc).Name()
		fileName := path.Base(file)
		funcName = strings.Split(funcName, ".")[1]
		fmt.Fprintf(panicInfo, "funcname:%s filename:%s LineNo:%d\n", funcName, fileName, lineNo)
	}

	return panicInfo.String()
}
