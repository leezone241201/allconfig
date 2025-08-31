package main

import (
	"fmt"

	"github.com/leezone241201/allconfig/config"
	"github.com/leezone241201/allconfig/middleware/db"
)

func main() {
	conf := config.NewConfig()
	fmt.Printf("%+v\n", db.NewGormMysqlManager(conf.MysqlNodes))
}
