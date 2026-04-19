# Site Health Monitor

[English](README_EN.md) | 中文

实时监控 OpenClash 代理节点健康状态的可视化 Dashboard，支持多地区节点分类、延迟统计与故障告警。

<p align="center">
  <img src="https://img.shields.io/github/v/release/TeddyXiong79/site-health-monitor?style=flat-square" alt="Release">
  <img src="https://img.shields.io/github/license/TeddyXiong79/site-health-monitor?style=flat-square" alt="License">
  <img src="https://img.shields.io/badge/Go-1.26+-00ADD8?style=flat-square&logo=go" alt="Go">
  <img src="https://img.shields.io/badge/Platform-amd64%20%7C%20arm64-blue?style=flat-square" alt="Platform">
</p>

## ✨ 功能特性

- **节点健康监控** — 自动抓取 OpenClash API 数据，实时展示所有代理节点的延迟状态
- **智能分类** — 按地区（香港/新加坡/台湾/日本/美国等）和延迟等级自动分组
- **实时统计** — 健康度百分比、高速/正常/低延迟/故障节点数量一目了然
- **赛博朋克 Dashboard** — Matrix Rain 背景特效、数字波浪动画、移动端自动优化
- **代理切换** — 点击节点名即可切换 `🔰国外流量` 策略组的当前节点，实时反馈（组名硬编码，详见下文配置说明）
- **自动刷新** — 可配置定时刷新（默认**300秒**），支持手动触发延迟检测
- **对外 API** — Bearer Token 认证的 RESTful API，方便接入 Claw/AI 等系统分析
- **数据缓存** — 10 秒 TTL 缓存层，避免高并发时对上游 OpenClash 产生重复请求
- **配置持久化** — Docker Volume 自动持久化，容器重启配置不丢失
- **容器化部署** — Docker 多平台镜像（amd64/arm64），scratch 极简基础镜像

## 🏗️ 架构设计

```
┌──────────────┐    ┌──────────────────────────────┐    ┌───────────────┐
│   Browser    │───▶│     Go HTTP Server            │───▶│  OpenClash    │
│  Dashboard   │◀───│     (gorilla/mux)             │◀───│  路由器 API    │
└──────────────┘    │                              │    └───────────────┘
                    │  ┌─────────┐ ┌────────────┐  │
┌──────────────┐    │  │ 缓存层   │ │ 限流器      │  │
│  Claw/AI     │───▶│  │ 10s TTL │ │ Per-IP     │  │
│  MCP 客户端   │◀───│  └─────────┘ └────────────┘  │
└──────────────┘    │                              │
                    │  ┌──────────────────────────┐ │
                    │  │ /data/config.json        │ │
                    │  │ Docker Volume 持久化      │ │
                    │  └──────────────────────────┘ │
                    └──────────────────────────────┘
```

### 技术栈

| 组件 | 技术 | 说明 |
|------|------|------|
| Web 框架 | [gorilla/mux](https://github.com/gorilla/mux) | HTTP 路由 |
| 配置管理 | [spf13/viper](https://github.com/spf13/viper) | JSON 配置读写 |
| 限流 | [golang.org/x/time/rate](https://pkg.go.dev/golang.org/x/time/rate) | Per-IP Token Bucket |
| 前端 | 原生 HTML/CSS/JS | 赛博朋克风格，零依赖 |
| 容器 | scratch 基础镜像 | 极简，仅包含二进制 + 模板 + CA 证书 |

### 数据流

1. **Dashboard/API 请求** → 缓存层检查（10s TTL）
2. **缓存未命中** → 向 OpenClash API 发起 HTTP 请求获取节点数据
3. **数据处理** → 按地区分类、延迟分级、排序统计
4. **返回响应** → Dashboard 渲染或 JSON API 返回

## 🚀 快速开始

### Docker 运行（推荐）

```bash
# 一键启动（配置自动持久化，无需额外操作）
docker run -d \
  --name site-health-monitor \
  --restart always \
  -p 9099:9099 \
  ghcr.io/teddyxiong79/site-health-monitor:latest
```

首次启动后访问 `http://localhost:9099/`，在 Dashboard 底部配置数据源信息即可。配置自动保存到 Docker Volume，容器重启不丢失。

```bash
# 彻底删除容器及配置数据
docker rm -v site-health-monitor
```

### Docker Compose（推荐生产环境）

```yaml
services:
  site-health-monitor:
    image: ghcr.io/teddyxiong79/site-health-monitor:latest
    container_name: site-health-monitor
    ports:
      - "9099:9099"
    volumes:
      - monitor-data:/data
    restart: unless-stopped

volumes:
  monitor-data:
```

```bash
# 启动
docker compose up -d

# 彻底清除（容器 + 配置数据）
docker compose down -v
```

### 二进制运行

```bash
# 下载对应平台的二进制文件
chmod +x site-health-monitor
./site-health-monitor
```

服务启动后访问：http://localhost:9099/

## ⚙️ 配置说明

### 🚨 OpenClash 前置要求（重要）

本程序**强制依赖**一个名为 **`🔰国外流量`** 的 OpenClash 策略组（Selector 类型，带 emoji 前缀）。该组名**硬编码**于 `fetcher.go`，以下功能都围绕它工作：

- **延迟刷新**（`POST /api/refresh`）— 触发 `🔰国外流量` 组内所有节点的实时延迟测试
- **代理切换**（`POST /api/switch`）— 修改 `🔰国外流量` 组当前选中的节点

> ⚠️ 如果你的 OpenClash 配置里没有这个组，或者名字不一样（比如 `🚀节点选择` / `Proxy`），这两个接口会返回 **404 分组不存在**。Dashboard 的节点数据展示不受影响，但"点击切换"和"刷新延迟"功能会失效。
>
> 如需使用不同的组名，请修改 `fetcher.go:206` 和 `fetcher.go:256` 两处后重新编译。

### Dashboard 配置项

首次启动后，在 Dashboard 界面填写以下配置：

| 参数 | 说明 | 默认值 |
|------|------|--------|
| 数据源地址 | OpenClash 所在的 IP | - |
| 数据源端口 | OpenClash API 端口 | `9090` |
| API 密钥 | OpenClash 控制面板密钥 | - |
| 刷新时间 | Dashboard 自动刷新间隔（秒） | `300` |

配置文件 `config.json`：

```json
{
  "api_address": "",
  "api_secret": "",
  "api_source_port": "9090",
  "port": "9099",
  "refresh_seconds": 300
}
```

> 💡 修改配置时密钥字段留空则保留原密钥不变，无需每次重新输入。

## 📡 API 接口

### 内部接口（Dashboard 同源调用）

| 方法 | 路径 | 说明 | 认证 |
|------|------|------|------|
| GET | `/` | Dashboard 页面 | 无 |
| GET | `/api/data` | 获取当前节点数据和统计 | 无 |
| GET | `/api/health` | 健康检查 | 无 |
| POST | `/api/config` | 保存配置 | 有密钥时需 Bearer Token |
| POST | `/api/test` | 测试 OpenClash 连接 | 有密钥时需 Bearer Token |
| POST | `/api/refresh` | 触发 `🔰国外流量` 组延迟检测（限流：10s/次） | 无 |
| POST | `/api/switch` | 切换 `🔰国外流量` 组的当前节点（限流：10s/次） | 有密钥时需 Bearer Token |

### 外部接口（需 Bearer Token 认证 + 限流）

所有外部接口需要在 Header 中携带：`Authorization: Bearer <API密钥>`

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/summary` | 获取统计摘要 |
| GET | `/api/nodes` | 获取所有节点列表 |
| GET | `/api/regions` | 获取所有地区及节点 |
| GET | `/api/regions/{code}/nodes` | 获取指定地区的节点 |
| GET | `/api/nodes/filter?region=&category=` | 按地区和分类过滤 |
| GET | `/api/nodes/{name}` | 获取指定节点详情 |

### 返回示例

```json
// GET /api/summary
{
  "total": 103,
  "fast": 20,
  "normal": 38,
  "high_latency": 21,
  "fault": 24,
  "healthy_count": 58,
  "healthy_pct": 56
}
```

## 📊 节点延迟分类

| 等级 | 延迟范围 | 说明 |
|------|----------|------|
| 🚀 高速 | ≤ 150ms | 优质节点 |
| ✅ 正常 | 151 ~ 240ms | 可用节点 |
| ⚠️ 低速 | 241 ~ 500ms | 高延迟节点 |
| ❌ 故障 | > 500ms 或 ≤ 0ms | 不可用节点（包括负延迟） |

## 🌍 地区分类

自动识别五大地区：**中国香港、新加坡、中国台湾、日本、美国**。未匹配的节点统一归入"其他地区"。

## 🔒 安全特性

- **Bearer Token 认证** — 外部 API 使用常量时间比较防止时序攻击
- **SSRF 防护** — 验证数据源地址，阻止回环地址和 URL 注入
- **响应体大小限制** — 上游响应限制 10MB，防止 OOM
- **Per-IP 限流** — 外部 API 和操作接口独立限流，支持反向代理 X-Forwarded-For
- **密钥脱敏** — Dashboard 不暴露完整密钥，仅显示脱敏版本
- **配置隔离** — Docker 镜像不打包敏感配置，运行时通过 Volume 持久化

## 🐳 镜像说明

| 镜像 | 说明 |
|------|------|
| `ghcr.io/teddyxiong79/site-health-monitor:latest` | 最新稳定版 |
| `ghcr.io/teddyxiong79/site-health-monitor:v1.6.2` | 指定版本 |

支持平台：`linux/amd64`、`linux/arm64`。

## 🛠️ 构建

```bash
# 本地构建
git clone https://github.com/TeddyXiong79/site-health-monitor.git
cd site-health-monitor
go build -o site-health-monitor .

# Docker 多平台构建
docker buildx build --platform=linux/amd64,linux/arm64 \
  -t ghcr.io/teddyxiong79/site-health-monitor:latest --push .
```

## 📝 版本历史

- **v1.6.2** — Bugfix：认证响应JSON化 + Token同步 + 延迟分类修正
  - 🐛 所有 401 认证失败响应从纯文本改为 JSON 格式（handlers.go 8 处），外部 API 调用方不再因解析崩溃
  - 🐛 Dashboard 修改密钥后同步更新隐藏的 raw-token 字段，避免后续操作仍用旧 Token 导致 401
  - 🐛 `categorizeDelay` 负延迟值（≤0）正确归类为 `fault`，不再落入 `normal`
  - 📦 默认刷新间隔从 120 秒调整为 **300 秒**
- **v1.6.1** — 安全加固 + 全面测试 + 文档同步
  - 🔒 `/api/config` 和 `/api/test` 增加 Bearer Token 认证（首次配置无密钥时允许免认证）
  - 🔒 `maskToken()` 修复：短密钥（≤4字符）不再明文暴露，统一返回 `***`
  - 🐛 `ValidateConfig` 修复：允许 API 密钥为空（兼容首次配置及无密钥 OpenClash），新增数据源地址非空校验
  - 📝 API 文档 (`docs/api.md`) 全面同步：延迟分类标准与代码对齐、删除已废弃的 `/api/token` 端点
  - ✅ 新增 40+ 单元测试，核心函数 100% 覆盖
  - 📦 提供 Windows AMD64 可执行文件下载
- **v1.6.0** — 安全加固 + 性能优化 + 配置持久化
  - 🔒 移除 Dashboard HTML 中的明文密钥暴露，引入 SafeConfig 脱敏渲染
  - 🔒 SSRF 防护（地址验证）、响应体大小限制（10MB）
  - 🔒 外部 API Bearer Token 认证 + 常量时间比较
  - 🔒 删除 /api/token 端点（安全风险）
  - ⚡ 新增 10 秒 TTL 数据缓存层，避免上游重复请求
  - ⚡ 限流器修复：正确提取 IP（去端口号）、支持 X-Forwarded-For/X-Real-IP
  - ⚡ /api/refresh 和 /api/switch 加限流（每 10 秒 1 次）
  - ⚡ 移动端自动禁用 Matrix Rain 动画，节省 CPU/电池
  - 🐛 优雅关闭改为 srv.Shutdown（等待请求完成）
  - 🐛 修复密钥"留空不修改"的前端承诺与后端行为不一致
  - 🐛 修复 fetchDataAndUpdate 不检查 HTTP 状态码导致数据清零
  - 🐛 修复倒计时重复触发和请求风暴风险
  - 🐛 修复 internalLimiter goroutine 泄漏
  - 🐛 修复缓存错误被缓存导致恢复延迟
  - 🐛 配置校验增强（端口范围 1-65535、刷新间隔 10-3600s）
  - 📦 配置持久化：Docker Volume 自动持久化，用户零操作
  - 📦 Dockerfile 修复：正确镜像版本、不打包密钥、添加 CA 证书
  - 📦 默认刷新间隔改为 **300 秒**
  - 📦 新增 .gitignore + config.json.example
- **v1.5.2** — 新增代理节点切换、204 兼容、刷新间隔默认 90 秒
- **v1.5.1** — 并发安全修复、Bug 优化、Docker 构建完善
- **v1.5.0** — 初始版本

## License

MIT
