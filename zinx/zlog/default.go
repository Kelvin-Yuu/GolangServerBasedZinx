package zlog

import (
	"context"
	"fmt"
	"zinx_server/zinx/ziface"
)

var LogInstance ziface.ILogger = new(DefaultLog)

type DefaultLog struct{}

func (log *DefaultLog) InfoF(format string, v ...interface{}) {
	StdLog.Infof(format, v...)
}

func (log *DefaultLog) ErrorF(format string, v ...interface{}) {
	StdLog.Errorf(format, v...)
}

func (log *DefaultLog) DebugF(format string, v ...interface{}) {
	StdLog.Debugf(format, v...)
}

func (log *DefaultLog) InfoFX(ctx context.Context, format string, v ...interface{}) {
	fmt.Println(ctx)
	StdLog.Infof(format, v...)
}

func (log *DefaultLog) ErrorFX(ctx context.Context, format string, v ...interface{}) {
	fmt.Println(ctx)
	StdLog.Errorf(format, v...)
}

func (log *DefaultLog) DebugFX(ctx context.Context, format string, v ...interface{}) {
	fmt.Println(ctx)
	StdLog.Debugf(format, v...)
}

func SetLogger(newlog ziface.ILogger) {
	LogInstance = newlog
}

func Ins() ziface.ILogger {
	return LogInstance
}
