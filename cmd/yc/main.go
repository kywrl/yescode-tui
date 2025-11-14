package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"yescode-tui/internal/api"
	"yescode-tui/internal/tui"
)

func main() {
	var (
		apiKeyFlag = flag.String("api-key", "", "YesCode API Key（可使用环境变量 YESCODE_API_KEY）")
		baseURL    = flag.String("base-url", "", "自定义 API Base URL（默认 https://co.yes.vg）")
	)
	flag.Parse()

	apiKey := strings.TrimSpace(*apiKeyFlag)
	if apiKey == "" {
		apiKey = strings.TrimSpace(os.Getenv("YESCODE_API_KEY"))
	}
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "缺少 API Key，请使用 --api-key 或设置环境变量 YESCODE_API_KEY")
		os.Exit(1)
	}

	var opts []api.Option
	if custom := strings.TrimSpace(*baseURL); custom != "" {
		opts = append(opts, api.WithBaseURL(custom))
	}

	client, err := api.NewClient(apiKey, opts...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化 API 客户端失败: %v\n", err)
		os.Exit(1)
	}

	program := tea.NewProgram(
		tui.NewModel(client),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(), // 启用鼠标支持
	)
	if err := program.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "程序运行失败: %v\n", err)
		os.Exit(1)
	}
}
