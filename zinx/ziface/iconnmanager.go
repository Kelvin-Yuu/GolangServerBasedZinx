package ziface

/*
连接管理模块抽象层
*/
type IConnManager interface {
	//添加链接
	Add(conn IConnection)

	//删除链接
	Remove(conn IConnection)

	//根据connID获取链接
	Get(connID uint64) (IConnection, error)

	//根据字符串connID获取链接
	Get2(connIdStr string) (IConnection, error)

	//得到当前链接总数
	Len() int

	//清除并终止所有链接
	ClearConn()

	//获取所有链接ID
	GetAllConnID() []uint64

	//获取所有链接ID字符串
	GetAllConnIdStr() []string

	//遍历所有连接
	Range(func(uint64, IConnection, interface{}) error, interface{}) error
	//遍历所有连接2
	Range2(func(string, IConnection, interface{}) error, interface{}) error
}
