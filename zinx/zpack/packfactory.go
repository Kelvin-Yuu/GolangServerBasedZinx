package zpack

import (
	"sync"
	"zinx_server/zinx/ziface"
)

/*
生成不同封包解包方法，【单例】
*/

var pack_once sync.Once

type pack_factory struct{}

var factoryInstance *pack_factory

func Factory() *pack_factory {
	pack_once.Do(func() {
		factoryInstance = new(pack_factory)
	})
	return factoryInstance
}

// 创建一个具体的拆包解包对象
func (f *pack_factory) NewPack(kind string) ziface.IDataPack {
	var dataPack ziface.IDataPack

	switch kind {
	case ziface.ZinxDataPack:
		dataPack = NewDataPack()
	case ziface.ZinxDataPackOld:
		//dataPack = NewDataPackLtv()
		//不实现小端LTV方法
	default:
		dataPack = NewDataPack()
	}
	return dataPack
}
