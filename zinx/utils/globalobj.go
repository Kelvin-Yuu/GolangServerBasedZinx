package utils

import (
	"encoding/json"
	"io/ioutil"
	"zinx_server/zinx/ziface"
)

/*
	存储一切有关框架的全局参数，供其他模块使用
*/

type GlobalObj struct {

	/*
		Server
	*/
	TcpServer ziface.IServer //当前Zinx全局的Server对象
	Host      string         //当前服务器主机监听的IP
	TCPPort   int            //当前服务器主机监听的端口号
	Name      string         //当前服务器的名称

	/*
		Zinx
	*/
	Version          string //当前Zinx的版本号
	MaxConn          int    //当前服务器主机允许的最大链接数
	MaxPackageSize   uint32 //当前框架数据包的最大值
	WorkerPoolSize   uint32 //当前业务工作Worker池的worker数量（Worker工作池的队列大小）
	MaxWorkerTaskLen uint32 //每个worker对应的消息队列的任务数量最大值
	// MaxWorkerTaskLen uint32 //框架允许用户最多开辟多少个Worker（限定条件）
}

/*
定义一个全局的对外GlobalObj
*/
var GlobalObject *GlobalObj

// 从zinx.json去加载用于自定义的参数
func (g *GlobalObj) Reload() {
	data, err := ioutil.ReadFile("conf/zinx.json")
	if err != nil {
		panic(err)
	}
	//将json文件数据解析到struct中
	err = json.Unmarshal(data, &GlobalObject)
	if err != nil {
		panic(err)
	}
}

// 提供一个init方法，初始化当前的GlobalObject
func init() {
	GlobalObject = &GlobalObj{
		Name:             "ZinxServerApp",
		Version:          "V1.0",
		TCPPort:          8999,
		Host:             "0.0.0.0",
		MaxConn:          12000,
		MaxPackageSize:   4096,
		WorkerPoolSize:   10,
		MaxWorkerTaskLen: 1024,
	}

	//应该尝试从conf/zinx.json去加载一些用户自定义的参数
	GlobalObject.Reload()
}
