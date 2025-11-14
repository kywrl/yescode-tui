# YesCode TUI

一个用于管理 YesCode API 服务的终端用户界面（TUI）工具。

## 功能特性

- **用户资料管理** - 查看账户信息、余额、订阅计划和消费统计
- **提供商管理** - 浏览和切换不同的 API 提供商
- **余额偏好设置** - 配置余额使用策略（优先订阅 / 仅按量付费）
- **实时刷新** - 自动更新用户资料信息
- **直观界面** - Material Design 风格，清晰易用

## 安装

### 通过 go install（推荐）

```bash
go install github.com/kywrl/yescode-tui/cmd/yc@latest
```

### 从源码构建

```bash
git clone https://github.com/kywrl/yescode-tui.git
cd yescode-tui
go install ./cmd/yc
```

## 快速开始

### 使用 API Key 启动

```bash
yc --api-key YOUR_API_KEY
```

### 通过环境变量配置

```bash
export YESCODE_API_KEY=YOUR_API_KEY
yc
```

### 自定义 API 端点

```bash
yc --api-key YOUR_API_KEY --base-url https://custom.api.url
```

## 键盘操作

### 标签页切换
- `Tab` / `Shift+Tab` - 前后切换标签页
- `1` / `2` / `3` - 直接跳转到指定标签页

### 导航操作
- `↑` `↓` 或 `k` `j` - 上下移动
- `←` `→` 或 `h` `l` - 切换焦点（提供商标签页）
- `Enter` - 确认选择
- `r` - 刷新当前视图

### 帮助
- `?` - 显示详细操作帮助弹窗
- `Esc` - 关闭帮助弹窗或退出程序
- `Ctrl+C` - 退出程序

## 鼠标操作

所有常用操作均支持鼠标：

- **点击标签页** - 直接切换到对应标签页
- **点击列表项** - 选择提供商或备选方案
- **滚轮滚动** - 滚动内容或移动选择
- **点击备选方案** - 在提供商标签页中点击右侧列表直接切换

## 界面预览

程序包含三个标签页：

1. **用户资料** - 显示账户信息和余额详情
2. **提供商** - 管理 API 提供商和备选方案
3. **余额使用偏好** - 配置余额使用策略

## 系统要求

- Go 1.24+ （仅构建时需要）
- 支持 ANSI 颜色的终端

## 许可证

MIT License

## 贡献

欢迎提交 Issue 和 Pull Request。
