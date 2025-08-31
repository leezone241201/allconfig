package main

import (
	"fmt"

	"github.com/leezone/allconfig/config"
	"github.com/leezone/allconfig/middleware/db"
)

func main() {
	conf := config.NewConfig()
	fmt.Printf("%+v\n", db.NewGormMysqlManager(conf.MysqlNodes))
}
