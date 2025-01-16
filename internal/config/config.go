package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	Analyzer  AnalyzerConfig `mapstructure:"analyzer"`
	Blocker   BlockerConfig  `mapstructure:"blocker"`
	Checker   CheckerConfig  `mapstructure:"checker"`
	Storage   StorageConfig  `mapstructure:"storage"`
	Profiling struct {
		Enabled bool `mapstructure:"enabled"`
		Port    int  `mapstructure:"port"`
	} `mapstructure:"profiling"`
}

type AnalyzerConfig struct {
	Network struct {
		IP struct {
			Enabled     bool   `mapstructure:"enabled"`
			Interface   string `mapstructure:"interface"`
			BufferSize  int32  `mapstructure:"buffer_size"`
			Promiscuous bool   `mapstructure:"promiscuous"`
		} `mapstructure:"ip"`
		DNS struct {
			Enabled bool `mapstructure:"enabled"`
			Port    int  `mapstructure:"port"`
		} `mapstructure:"dns"`
	} `mapstructure:"network"`
}

type BlockerConfig struct {
	IP struct {
		Enabled         bool     `mapstructure:"enabled"`
		DefaultDuration string   `mapstructure:"default_duration"`
		Whitelist       []string `mapstructure:"whitelist"`
		IPTables        bool     `mapstructure:"iptables"`
		NFTables        bool     `mapstructure:"nftables"`
	} `mapstructure:"ip"`
}

type CheckerConfig struct {
	IPDBPath string `mapstructure:"ipdb_path"`
	MMDBPath string `mapstructure:"mmdb_path"`
}

type StorageConfig struct {
	Type     string `mapstructure:"type"`
	Database struct {
		Driver string `mapstructure:"driver"`
		DSN    string `mapstructure:"dsn"`
	} `mapstructure:"database"`
}

var cfg *Config

func Init() error {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath("/etc/safepanel")
	viper.AddConfigPath("/usr/local/etc/safepanel")

	if err := viper.ReadInConfig(); err != nil {
		return err
	}

	cfg = &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return err
	}

	return nil
}

func Get() *Config {
	return cfg
}
