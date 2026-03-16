package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Elasticsearch ESConfig  `mapstructure:"elasticsearch" json:"elasticsearch" yaml:"elasticsearch"`
	Database      DBConfig  `mapstructure:"database" json:"database" yaml:"database"`
	App           AppConfig `mapstructure:"app" json:"app" yaml:"app"`
}

type ESConfig struct {
	IP        string `mapstructure:"ip" json:"ip" yaml:"ip"`
	Port      int    `mapstructure:"port" json:"port" yaml:"port"`
	Index     string `mapstructure:"index" json:"index" yaml:"index"`
	User      string `mapstructure:"user" json:"user" yaml:"user"`
	Password  string `mapstructure:"password" json:"password" yaml:"password"`
	TimeField string `mapstructure:"time_field" json:"time_field" yaml:"time_field"`
	TypeField string `mapstructure:"type_field" json:"type_field" yaml:"type_field"`
}

type DBConfig struct {
	Type         string `mapstructure:"type" json:"type" yaml:"type"`
	Host         string `mapstructure:"host" json:"host" yaml:"host"`
	Port         int    `mapstructure:"port" json:"port" yaml:"port"`
	User         string `mapstructure:"user" json:"user" yaml:"user"`
	Password     string `mapstructure:"password" json:"password" yaml:"password"`
	DBName       string `mapstructure:"dbname" json:"dbname" yaml:"dbname"`
	Schema       string `mapstructure:"schema" json:"schema" yaml:"schema"`
	ConnUrl      string `mapstructure:"conn_url" json:"conn_url" yaml:"conn_url"`
	DriverPath   string `mapstructure:"driver_path" json:"driver_path" yaml:"driver_path"`
	DriverClass  string `mapstructure:"driver_class" json:"driver_class" yaml:"driver_class"`
	DictTable    string `mapstructure:"dict_table" json:"dict_table" yaml:"dict_table"`
	DictCodeCol  string `mapstructure:"dict_code_col" json:"dict_code_col" yaml:"dict_code_col"`
	DictKeyCol   string `mapstructure:"dict_key_col" json:"dict_key_col" yaml:"dict_key_col"`
	DictValueCol string `mapstructure:"dict_value_col" json:"dict_value_col" yaml:"dict_value_col"`
}

type AppConfig struct {
	Debug      bool   `mapstructure:"debug" json:"debug" yaml:"debug"`
	ExportPath string `mapstructure:"export_path" json:"export_path" yaml:"export_path"`
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

func UpdateAndSaveConfig(configPath string, cfg *Config) error {
	v := viper.New()
	v.SetConfigFile(configPath)

	v.Set("elasticsearch", cfg.Elasticsearch)
	v.Set("database", cfg.Database)
	v.Set("app", cfg.App)

	if err := v.WriteConfig(); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	GlobalConfig = cfg
	return nil
}
