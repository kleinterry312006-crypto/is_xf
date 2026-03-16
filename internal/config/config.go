package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Elasticsearch ESConfig  `mapstructure:"elasticsearch"`
	Database      DBConfig  `mapstructure:"database"`
	App           AppConfig `mapstructure:"app"`
}

type ESConfig struct {
	Address   string `mapstructure:"address"`
	Index     string `mapstructure:"index"`
	TimeField string `mapstructure:"time_field"`
	TypeField string `mapstructure:"type_field"`
}

type DBConfig struct {
	Type         string `mapstructure:"type"`
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	User         string `mapstructure:"user"`
	Password     string `mapstructure:"password"`
	DBName       string `mapstructure:"dbname"`
	Schema       string `mapstructure:"schema"`
	ConnUrl      string `mapstructure:"conn_url"`
	DriverClass  string `mapstructure:"driver_class"`
	DictTable    string `mapstructure:"dict_table"`
	DictCodeCol  string `mapstructure:"dict_code_col"`
	DictKeyCol   string `mapstructure:"dict_key_col"`
	DictValueCol string `mapstructure:"dict_value_col"`
}

type AppConfig struct {
	Debug      bool   `mapstructure:"debug"`
	ExportPath string `mapstructure:"export_path"`
}

var GlobalConfig *Config

func LoadConfig(configPath string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(configPath)
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	GlobalConfig = &cfg
	return &cfg, nil
}
