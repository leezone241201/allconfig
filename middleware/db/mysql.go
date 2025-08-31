package db

import (
	"log"
	"time"

	"github.com/leezone/allconfig/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type newFunc func(string) (interface{}, error)
type node[T any] struct {
	info NodeInfo
	db   T
}

type MysqlManager[T any] struct {
	masters []node[T]
	slaves  []node[T]
}

func (m *MysqlManager[T]) GetMasterDB(selectFunc func(PollingContext) int) T {
	pollingContext := m.getPollingContext("master")
	return m.masters[selectFunc(pollingContext)].db
}

func (m *MysqlManager[T]) GetSlaveDB(selectFunc func(PollingContext) int) T {
	pollingContext := m.getPollingContext("slave")
	return m.masters[selectFunc(pollingContext)].db
}

func (m *MysqlManager[T]) Close() error {
	return nil
}

func (m *MysqlManager[T]) getPollingContext(nodeType string) PollingContext {
	pollingContext := PollingContext{}
	var rangeNodes []node[T]

	if nodeType == "master" {
		rangeNodes = m.masters
	} else {
		rangeNodes = m.slaves
	}

	for _, node := range rangeNodes {
		pollingContext.Nodes = append(pollingContext.Nodes, node.info)
	}

	return pollingContext
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
		masters: masters,
		slaves:  slaves,
	}
}

func NewGormMysqlManager(configs config.MysqlConfigs) DBManager[*gorm.DB] {
	return NewMysqlManager[*gorm.DB](configs, getGormDB)
}
