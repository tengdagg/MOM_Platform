// Copyright (c) 2026 DYCloud J.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package conf

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config 全局配置
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Log      LogConfig      `mapstructure:"log"`
	Guacd    GuacdConfig    `mapstructure:"guacd"`
	Security SecurityConfig `mapstructure:"security"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Mode         string `mapstructure:"mode"` // debug, release, test
	HttpPort     int    `mapstructure:"http_port"`
	RPCPort      int    `mapstructure:"rpc_port"`
	ReadTimeout  int    `mapstructure:"read_timeout"`  // 毫秒
	WriteTimeout int    `mapstructure:"write_timeout"` // 毫秒
	JWTSecret    string `mapstructure:"jwt_secret"`    // JWT密钥
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Driver          string `mapstructure:"driver"`
	Host            string `mapstructure:"host"`
	Port            int    `mapstructure:"port"`
	Database        string `mapstructure:"database"`
	Username        string `mapstructure:"username"`
	Password        string `mapstructure:"password"`
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`
	MaxOpenConns    int    `mapstructure:"max_open_conns"`
	ConnMaxLifetime int    `mapstructure:"conn_max_lifetime"` // 秒
}

// RedisConfig Redis配置
type RedisConfig struct {
	Host        string `mapstructure:"host"`
	Port        int    `mapstructure:"port"`
	Password    string `mapstructure:"password"`
	DB          int    `mapstructure:"db"`
	PoolSize    int    `mapstructure:"pool_size"`
	MinIdleConn int    `mapstructure:"min_idle_conn"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string `mapstructure:"level"`
	Filename   string `mapstructure:"filename"`
	MaxSize    int    `mapstructure:"max_size"` // MB
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"` // days
	Compress   bool   `mapstructure:"compress"`
	Console    bool   `mapstructure:"console"`
}

// GuacdConfig Guacamole代理守护进程配置 (用于Windows RDP远程连接)
type GuacdConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	CredentialEncryptionKey string `mapstructure:"credential_encryption_key"`
	K8sEncryptionKey        string `mapstructure:"k8s_encryption_key"`
}

// GetGuacdAddr 获取Guacd地址
func (c *GuacdConfig) GetGuacdAddr() string {
	host := c.Host
	port := c.Port
	if host == "" {
		host = "127.0.0.1"
	}
	if port == 0 {
		port = 4822
	}
	return fmt.Sprintf("%s:%d", host, port)
}

var globalConfig *Config

// Load 加载配置
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// 设置配置文件
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	// 环境变量前缀
	v.SetEnvPrefix("mom")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// 设置默认值（Docker 环境下可纯靠环境变量启动，无需配置文件）
	v.SetDefault("server.mode", "release")
	v.SetDefault("server.http_port", 9876)
	v.SetDefault("server.rpc_port", 9090)
	v.SetDefault("server.read_timeout", 60000)
	v.SetDefault("server.write_timeout", 60000)
	v.SetDefault("server.jwt_secret", "your-secret-key-change-in-production")
	v.SetDefault("database.driver", "mysql")
	v.SetDefault("database.host", "127.0.0.1")
	v.SetDefault("database.port", 3306)
	v.SetDefault("database.database", "mom")
	v.SetDefault("database.username", "root")
	v.SetDefault("database.password", "")
	v.SetDefault("database.max_idle_conns", 10)
	v.SetDefault("database.max_open_conns", 100)
	v.SetDefault("database.conn_max_lifetime", 3600)
	v.SetDefault("redis.host", "127.0.0.1")
	v.SetDefault("redis.port", 6379)
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)
	v.SetDefault("redis.pool_size", 10)
	v.SetDefault("redis.min_idle_conn", 5)
	v.SetDefault("guacd.host", "127.0.0.1")
	v.SetDefault("guacd.port", 4822)
	v.SetDefault("security.credential_encryption_key", "")
	v.SetDefault("security.k8s_encryption_key", "")
	v.SetDefault("log.level", "info")
	v.SetDefault("log.filename", "logs/app.log")
	v.SetDefault("log.max_size", 100)
	v.SetDefault("log.max_backups", 10)
	v.SetDefault("log.max_age", 30)
	v.SetDefault("log.compress", true)
	v.SetDefault("log.console", true)

	// 读取配置文件（文件不存在时使用默认值 + 环境变量）
	if err := v.ReadInConfig(); err != nil {
		fmt.Printf("[配置] 配置文件 %s 未找到，使用默认值 + 环境变量\n", configPath)
	}

	// 解析配置
	config := &Config{}
	if err := v.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	globalConfig = config
	return config, nil
}

// Get 获取全局配置
func Get() *Config {
	return globalConfig
}

// GetDSN 获取数据库连接字符串
func (c *DatabaseConfig) GetDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true&loc=Local&multiStatements=true",
		c.Username,
		c.Password,
		c.Host,
		c.Port,
		c.Database,
	)
}

// GetRedisAddr 获取Redis地址
func (c *RedisConfig) GetRedisAddr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}
