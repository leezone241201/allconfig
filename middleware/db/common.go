package db

type NodeInfo struct {
	Host string
	//TODO 后续可以加入负载等信息
}

type PollingContext struct {
	Nodes []NodeInfo
}

type DBManager[T any] interface {
	GetMasterDB(func(PollingContext) int) T
	GetSlaveDB(func(PollingContext) int) T
	Close() error
}
