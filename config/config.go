package config

import (
	"fmt"
	"log"

	"github.com/spf13/viper"
)

type Config struct {
	MysqlNodes MysqlConfigs `mapstructure:"mysql"`
}

type MysqlConfigs []MysqlConfig

type MysqlConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Database string `mapstructure:"database"`
	Charset  string `mapstructure:"charset"`
	Role     string `mapstructure:"role"`
}

func (m MysqlConfig) Dsn() string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?charset=%s&&parseTime=True&loc=Local",
		m.User,
		m.Password,
		m.Host,
		m.Port,
		m.Database,
		m.Charset,
	)
}

func NewConfig() *Config {
	v := viper.New()
	v.SetConfigName("conf")
	v.SetConfigType("yaml")
	v.AddConfigPath("./config")

	err := v.ReadInConfig()
	if err != nil {
		log.Fatalf("Fatal error config file: %s \n", err)
	}

	setDefault(v)
	v.AutomaticEnv()

	config := new(Config)
	err = v.Unmarshal(config)
	if err != nil {
		log.Fatalf("Unable to decode into struct, %v", err)
	}
	return config
}

func NewConfigByPath(path string) (*Config, error) {
	cfg := &Config{}
	return cfg, nil
}

func setDefault(v *viper.Viper) {
	v.SetDefault("mysql", []MysqlConfig{
		{
			Host:     "localhost",
			Port:     3306,
			User:     "example",
			Password: "example",
			Database: "mydb",
			Charset:  "utf8mb4",
			Role:     "master",
		},
	})
}
