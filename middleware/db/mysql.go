package db

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/leezone241201/allconfig/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type newFunc func(string) (interface{}, error)
type node[T any] struct {
	info NodeInfo
	db   T
}

// 当前没有做并发保护,在创建阶段应该注册好所有信息,运行阶段不允许变更
type MysqlManager[T any] struct {
	mu               sync.RWMutex           // 保护下面的变量
	masters          []node[T]              // master节点列表
	slaves           []node[T]              // slave节点列表
	currentSlave     int                    // 记录当前使用的slave节点
	currentMaster    int                    // 记录当前使用的master节点
	balanceFuncs     map[string]BalanceFunc // 注册的负载均衡函数
	balanceFuncIndex []string               // 负载均衡函数的调用顺序
	defaultFunc      BalanceFunc            // 默认的负载均衡函数,为轮询,在其他轮询算法失效时保证可以实现负载均衡
}

func (m *MysqlManager[T]) GetMasterDB() T {
	m.mu.Lock()
	defer m.mu.Unlock()
	balanceContext := m.getBalanceContext("master")
	db, index := m.chooseDB(balanceContext)
	m.currentMaster = index
	return db
}

func (m *MysqlManager[T]) GetSlaveDB() T {
	m.mu.Lock()
	defer m.mu.Unlock()
	balanceContext := m.getBalanceContext("slave")
	db, index := m.chooseDB(balanceContext)
	m.currentSlave = index
	return db
}

func (m *MysqlManager[T]) chooseDB(ctx BalanceContext) (T, int) {
	for _, fnName := range m.balanceFuncIndex {
		if fn, ok := m.balanceFuncs[fnName]; ok {
			if index, valid := fn(ctx); valid {
				return m.masters[index].db, index
			}
		}
	}
	// 所有注册的负载均衡函数都失效,使用默认的轮询
	index, _ := m.defaultFunc(ctx)
	return m.masters[index].db, index
}

func (m *MysqlManager[T]) Close() error {
	return nil
}

func (m *MysqlManager[T]) RegisterBalanceFunc(name string, fn BalanceFunc) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.balanceFuncs == nil {
		m.balanceFuncs = make(map[string]BalanceFunc)
		m.balanceFuncIndex = make([]string, 0)
	}

	if _, ok := m.balanceFuncs[name]; ok {
		return ErrBalanceFuncExist
	}

	m.balanceFuncs[name] = fn
	m.balanceFuncIndex = append(m.balanceFuncIndex, name)
	return nil
}

func (m *MysqlManager[T]) RemoveBalanceFunc(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.balanceFuncs[name]; ok {
		delete(m.balanceFuncs, name)
		for i, fnName := range m.balanceFuncIndex {
			if fnName == name {
				lastIndex := len(m.balanceFuncIndex) - 1
				m.balanceFuncIndex = append(m.balanceFuncIndex[:i], m.balanceFuncIndex[i+1:]...)
				m.balanceFuncIndex[lastIndex] = "" // clear the last element
				break
			}
		}
	}
}

func (m *MysqlManager[T]) getBalanceContext(nodeType string) BalanceContext {
	balanceContext := BalanceContext{}
	var rangeNodes []node[T]
	var currentNode int

	if nodeType == "master" {
		rangeNodes = m.masters
		currentNode = m.currentMaster
	} else {
		rangeNodes = m.slaves
		currentNode = m.currentSlave
	}

	for _, node := range rangeNodes {
		balanceContext.Nodes = append(balanceContext.Nodes, node.info)
	}
	balanceContext.CurrentNode = currentNode
	return balanceContext
}

func getGormDB(dsn string) (interface{}, error) {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	gormDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	var errCount int
	for err := gormDB.Ping(); err != nil; errCount++ {
		if errCount < 3 {
			time.Sleep(time.Duration(errCount*5) * time.Second)
			continue
		}
		log.Fatalf("mysql connect ping failed, err:%v", err)
	}

	gormDB.SetMaxIdleConns(10)
	gormDB.SetMaxOpenConns(100)
	return db, nil
}

func NewMysqlManager[T any](configs config.MysqlConfigs, newDBFunc newFunc) DBManager[T] {
	var masters = make([]node[T], 0)
	var slaves = make([]node[T], 0)

	for _, config := range configs {
		db, err := newDBFunc(config.Dsn())
		fmt.Println(config.Dsn())
		if err != nil {
			log.Fatalf("init mysql failed, err:%v", err)
			continue
		}

		node := node[T]{
			info: NodeInfo{
				Host: config.Host,
			},
			db: db.(T),
		}

		if config.Role == "master" {
			masters = append(masters, node)
		} else {
			slaves = append(slaves, node)
		}
	}

	return &MysqlManager[T]{
		masters:     masters,
		slaves:      slaves,
		defaultFunc: defualtBalanceFunc,
	}
}

func NewGormMysqlManager(configs config.MysqlConfigs) DBManager[*gorm.DB] {
	return NewMysqlManager[*gorm.DB](configs, getGormDB)
}
