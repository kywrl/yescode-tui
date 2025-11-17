package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"yescode-tui/internal/api"
	"yescode-tui/internal/logger"
	"yescode-tui/internal/tui"
)

func main() {
	var (
		apiKeyFlag = flag.String("api-key", "", "YesCode API Key（可使用环境变量 YESCODE_API_KEY）")
		baseURL    = flag.String("base-url", "", "自定义 API Base URL（默认 https://co.yes.vg）")
		debug      = flag.Bool("debug", false, "启用调试日志（日志文件：/tmp/yescode-tui.log）")
		logFile    = flag.String("log-file", "", "自定义日志文件路径（仅在 --debug 时有效）")
	)
	flag.Parse()

	// Initialize logger
	if err := logger.Init(*debug, *logFile); err != nil {
		fmt.Fprintf(os.Stderr, "初始化日志失败: %v\n", err)
		os.Exit(1)
	}
	logger.Info("yescode-tui starting")

	apiKey := strings.TrimSpace(*apiKeyFlag)
	if apiKey == "" {
		apiKey = strings.TrimSpace(os.Getenv("YESCODE_API_KEY"))
	}
	if apiKey == "" {
		logger.Error("API key not provided")
		fmt.Fprintln(os.Stderr, "缺少 API Key，请使用 --api-key 或设置环境变量 YESCODE_API_KEY")
		os.Exit(1)
	}

	var opts []api.Option
	if custom := strings.TrimSpace(*baseURL); custom != "" {
		logger.Debug("Using custom base URL: %s", custom)
		opts = append(opts, api.WithBaseURL(custom))
	}

	client, err := api.NewClient(apiKey, opts...)
	if err != nil {
		logger.Error("Failed to initialize API client: %v", err)
		fmt.Fprintf(os.Stderr, "初始化 API 客户端失败: %v\n", err)
		os.Exit(1)
	}
	logger.Debug("API client initialized successfully")

	program := tea.NewProgram(
		tui.NewModel(client),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(), // 启用鼠标支持
	)
	logger.Info("Starting Bubble Tea program")
	if err := program.Start(); err != nil {
		logger.Error("Program failed: %v", err)
		fmt.Fprintf(os.Stderr, "程序运行失败: %v\n", err)
		os.Exit(1)
	}
	logger.Info("Program exited normally")
}
