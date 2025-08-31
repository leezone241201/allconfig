package db

import "errors"

var ErrBalanceFuncExist = errors.New("balance function already exists")

type NodeInfo struct {
	Host string
	//TODO 后续可以加入负载等信息
}

type BalanceContext struct {
	Nodes       []NodeInfo
	CurrentNode int
}

type BalanceFunc func(BalanceContext) (int, bool)

type DBManager[T any] interface {
	GetMasterDB() T                                // 获取一个主库连接
	GetSlaveDB() T                                 // 获取一个从库连接
	Close() error                                  // TODO 关闭所有连接
	RegisterBalanceFunc(string, BalanceFunc) error // 向db管理器注册负载均衡函数
	RemoveBalanceFunc(string)                      // 移除db管理器注册的负载均衡函数
}

func defualtBalanceFunc(ctx BalanceContext) (int, bool) {
	return (ctx.CurrentNode + 1) % len(ctx.Nodes), true
}
