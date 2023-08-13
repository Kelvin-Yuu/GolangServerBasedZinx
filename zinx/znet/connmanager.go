package znet

import (
	"errors"
	"strconv"
	"zinx_server/zinx/ziface"
	"zinx_server/zinx/zlog"
	"zinx_server/zinx/zutils"
)

/*
连接管理模块
*/
type ConnManager struct {
	connections zutils.ShardLockMaps
}

// 创建当前链接的方法
func NewConnManager() *ConnManager {
	return &ConnManager{
		connections: zutils.NewShardLockMaps(),
	}
}

// 添加链接
func (connMgr *ConnManager) Add(conn ziface.IConnection) {
	// 将conn连接加到ConnManager中
	connMgr.connections.Set(conn.GetConnIdStr(), conn)
	zlog.Ins().InfoF("connection add to ConnManager successfully: conn num = %d", connMgr.Len())
}

// 删除链接
func (connMgr *ConnManager) Remove(conn ziface.IConnection) {
	//删除连接信息
	connMgr.connections.Remove(conn.GetConnIdStr())
	zlog.Ins().InfoF("connection Remove ConnID=%d successfully: conn num = %d", conn.GetConnID(), connMgr.Len())
}

// 根据connID获取链接
func (connMgr *ConnManager) Get(connID uint64) (ziface.IConnection, error) {
	strConnId := strconv.FormatUint(connID, 10)
	if conn, ok := connMgr.connections.Get(strConnId); ok {
		return conn.(ziface.IConnection), nil
	}

	return nil, errors.New("connection not found")
}

// 根据字符串connID获取链接
func (connMgr *ConnManager) Get2(connIdStr string) (ziface.IConnection, error) {
	if conn, ok := connMgr.connections.Get(connIdStr); ok {
		return conn.(ziface.IConnection), nil
	}

	return nil, errors.New("connection not found")
}

// 得到当前链接总数
func (connMgr *ConnManager) Len() int {
	length := connMgr.connections.Count()

	return length
}

// 清除并终止所有链接
func (connMgr *ConnManager) ClearConn() {
	cb := func(key string, val interface{}, exists bool) bool {
		if conn, ok := val.(ziface.IConnection); ok {
			conn.Stop()
			return true
		}
		return false
	}

	for item := range connMgr.connections.IterBuffered() {
		connMgr.connections.RemoveCb(item.Key, cb)
	}

	zlog.Ins().InfoF("Clear All Connections successfully: conn num = %d", connMgr.Len())
}

// 获取所有链接ID
func (connMgr *ConnManager) GetAllConnID() []uint64 {
	strConnIdList := connMgr.connections.Keys()
	ids := make([]uint64, 0, len(strConnIdList))

	for _, strId := range strConnIdList {
		connId, err := strconv.ParseUint(strId, 10, 64)
		if err == nil {
			ids = append(ids, connId)
		} else {
			zlog.Ins().InfoF("GetAllConnID Id: %d, error: %v", connId, err)
		}
	}
	return ids
}

func (connMgr *ConnManager) GetAllConnIdStr() []string {
	return connMgr.connections.Keys()
}

// 遍历所有连接
func (connMgr *ConnManager) Range(cb func(uint64, ziface.IConnection, interface{}) error, args interface{}) (err error) {
	connMgr.connections.IterCb(func(key string, v interface{}) {
		conn, _ := v.(ziface.IConnection)
		connId, _ := strconv.ParseUint(key, 10, 64)
		err = cb(connId, conn, args)
		if err != nil {
			zlog.Ins().InfoF("Range key: %v, v: %v, error: %v", key, v, err)
		}
	})
	return err
}

func (connMgr *ConnManager) Range2(cb func(string, ziface.IConnection, interface{}) error, args interface{}) (err error) {

	connMgr.connections.IterCb(func(key string, v interface{}) {
		conn, _ := v.(ziface.IConnection)
		err = cb(conn.GetConnIdStr(), conn, args)
		if err != nil {
			zlog.Ins().InfoF("Range2 key: %v, v: %v, error: %v", key, v, err)
		}
	})

	return err
}
