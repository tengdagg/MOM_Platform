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

package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/ydcloud-dy/mom/cmd/root"
	versionPkg "github.com/ydcloud-dy/mom/cmd/version"
	"github.com/ydcloud-dy/mom/internal/biz"
	"github.com/ydcloud-dy/mom/internal/conf"
	dataPkg "github.com/ydcloud-dy/mom/internal/data"
	"github.com/ydcloud-dy/mom/internal/migration"
	"github.com/ydcloud-dy/mom/internal/server"
	"github.com/ydcloud-dy/mom/internal/service"
	rbacservice "github.com/ydcloud-dy/mom/internal/service/rbac"
	appLogger "github.com/ydcloud-dy/mom/pkg/logger"
	"go.uber.org/zap"
)

// 全局变量，用于在服务器生命周期内保持连接
var (
	globalData       *dataPkg.Data
	globalRedis      *dataPkg.Redis
	globalHTTPServer *server.HTTPServer
)

var Cmd = &cobra.Command{
	Use:   "server",
	Short: "启动服务",
	Long:  `启动 mom HTTP 服务器`,
	PreRun: func(cmd *cobra.Command, args []string) {
		// 从命令行参数覆盖配置
		if mode := viper.GetString("mode"); mode != "" {
			viper.Set("server.mode", mode)
		}
		if logLevel := viper.GetString("log-level"); logLevel != "" {
			viper.Set("log.level", logLevel)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		// 加载配置
		cfg, err := runServer()
		if err != nil {
			fmt.Printf("启动服务失败: %v\n", err)
			os.Exit(1)
		}

		// 等待中断信号
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit

		fmt.Println("\n正在关闭服务...")
		ctx := context.Background()
		if err := stopServer(ctx, cfg); err != nil {
			fmt.Printf("关闭服务失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("服务已关闭")
	},
}

func init() {
	root.Cmd.AddCommand(Cmd)
}

func runServer() (*conf.Config, error) {
	// 加载配置
	cfg, err := conf.Load(root.GetConfigFile())
	if err != nil {
		return nil, fmt.Errorf("加载配置失败: %w", err)
	}

	// 初始化日志
	logCfg := &appLogger.Config{
		Level:      cfg.Log.Level,
		Filename:   cfg.Log.Filename,
		MaxSize:    cfg.Log.MaxSize,
		MaxBackups: cfg.Log.MaxBackups,
		MaxAge:     cfg.Log.MaxAge,
		Compress:   cfg.Log.Compress,
		Console:    cfg.Log.Console,
	}
	if err := appLogger.Init(logCfg); err != nil {
		return nil, fmt.Errorf("初始化日志失败: %w", err)
	}
	defer appLogger.Sync()

	appLogger.Info("服务启动中...",
		zap.String("version", versionPkg.Version),
		zap.String("mode", cfg.Server.Mode),
	)

	// 初始化数据层
	data, err := dataPkg.NewData(cfg)
	if err != nil {
		return nil, fmt.Errorf("初始化数据层失败: %w", err)
	}
	globalData = data // 保存到全局变量，防止被垃圾回收

	// 初始化Redis
	redis, err := dataPkg.NewRedis(cfg)
	if err != nil {
		return nil, fmt.Errorf("初始化Redis失败: %w", err)
	}
	globalRedis = redis // 保存到全局变量

	// 初始化验证码存储（使用 Redis）
	rbacservice.InitCaptchaStore(redis.Get())

	// 初始化业务层
	biz := biz.NewBiz(data, redis)

	// 初始化服务层
	svc := service.NewService(biz)

	// 运行数据库迁移（替代 autoMigrate + initDefaultData）
	if err := migration.Run(cfg.Database.GetDSN()); err != nil {
		return nil, fmt.Errorf("数据库迁移失败: %w", err)
	}

	// 创建缓存工具
	cache := dataPkg.NewCache(redis.Get())

	// 初始化HTTP服务器
	httpServer := server.NewHTTPServer(cfg, svc, data.DB(), cache)
	globalHTTPServer = httpServer // 保存到全局变量

	// 启动服务器
	go func() {
		if err := httpServer.Start(); err != nil && err != http.ErrServerClosed {
			appLogger.Fatal("HTTP服务器启动失败", zap.Error(err))
		}
	}()

	// 打印启动信息
	printStartupInfo(cfg)

	return cfg, nil
}

func stopServer(ctx context.Context, cfg *conf.Config) error {
	appLogger.Info("服务正在关闭...")

	// 停止HTTP服务器
	if globalHTTPServer != nil {
		if err := globalHTTPServer.Stop(ctx); err != nil {
			appLogger.Error("停止HTTP服务器失败", zap.Error(err))
		}
	}

	// 关闭数据库连接
	if globalData != nil {
		if err := globalData.Close(); err != nil {
			appLogger.Error("关闭数据库连接失败", zap.Error(err))
		}
	}

	// 关闭Redis连接
	if globalRedis != nil {
		if err := globalRedis.Close(); err != nil {
			appLogger.Error("关闭Redis连接失败", zap.Error(err))
		}
	}

	return nil
}

func printStartupInfo(cfg *conf.Config) {
	listenAddr := fmt.Sprintf("%s:%d", "0.0.0.0", cfg.Server.HttpPort)
	displayAddr := fmt.Sprintf("%s:%d", "127.0.0.1", cfg.Server.HttpPort)

	fmt.Println()
	fmt.Println("========================================")
	fmt.Println("       mom 运维管理平台启动成功")
	fmt.Println("========================================")
	fmt.Printf("版本:     %s\n", versionPkg.Version)
	fmt.Printf("模式:     %s\n", cfg.Server.Mode)
	fmt.Printf("监听地址: http://%s\n", listenAddr)
	fmt.Printf("健康检查: http://%s/health\n", displayAddr)
	fmt.Printf("API文档:  http://%s/swagger/index.html\n", displayAddr)
	fmt.Println("========================================")
	fmt.Println()
}
