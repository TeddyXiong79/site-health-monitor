# Site Health Monitor

实时监控 OpenClash 代理节点健康状态的可视化 Dashboard，支持多地区节点分类、延迟统计与故障告警。

## 功能特性

- **节点健康监控** — 自动抓取 OpenClash API 数据，实时展示所有代理节点的延迟状态
- **智能分类** — 按地区（香港/新加坡/台湾/日本/美国等）和延迟等级（高速/正常/低延迟/故障）自动分组
- **实时统计** — 健康度百分比、高速/正常/低延迟/故障节点数量一目了然
- **Dashboard 可视化** — 赛博朋克风格界面，数字动画、Matrix 背景特效
- **刷新控制** — 自动定时刷新 + 手动触发 OpenClash 延迟检测
- **对外 API** — 支持 Bearer Token 认证的 RESTful API，方便接入其他系统
- **容器化部署** — Docker 多平台镜像（amd64/arm64），scratch 极简基础镜像

## 架构

```
┌──────────────┐    ┌──────────────────┐    ┌───────────────┐
│   Browser   │───▶│  Go HTTP Server  │───▶│  OpenClash    │
│  Dashboard   │◀───│  (Mux Router)   │◀───│  Control Panel │
└──────────────┘    └──────────────────┘    └───────────────┘
                           │
                     ┌─────┴─────┐
                     │ Rate Limit │  Per-IP 限流
                     │  Cleanup   │  240min 过期自动清理
                     └───────────┘
```

## 快速开始

### Docker 运行（推荐）

```bash
# 拉取并运行（使用默认空配置，首次需在 Dashboard 配置）
docker run -d \
  --name site-health-monitor \
  -p 9099:9099 \
  ghcr.io/teddyxiong79/site-health-monitor:latest

# 挂载自定义配置文件
docker run -d \
  --name site-health-monitor \
  -p 9099:9099 \
  -v /path/to/config.json:/config.json \
  ghcr.io/teddyxiong79/site-health-monitor:latest
```

### Docker Compose

```yaml
services:
  site-health-monitor:
    image: ghcr.io/teddyxiong79/site-health-monitor:latest
    container_name: site-health-monitor
    ports:
      - "9099:9099"
    volumes:
      - ./config.json:/config.json
    restart: unless-stopped
```

### 二进制运行

```bash
# 下载对应平台的二进制文件
# Linux/macOS
chmod +x site-health-monitor
./site-health-monitor

# Windows
site-health-monitor.exe
```

服务启动后访问：http://localhost:9099/

## 配置说明

首次启动后，在 Dashboard 界面填写以下配置：

| 参数 | 说明 | 示例 |
|------|------|------|
| 数据源地址 | OpenClash 所在的 IP 或域名 | `192.168.66.251` |
| 数据源端口 | OpenClash API 端口，默认 `9090` | `9090` |
| API 密钥 | OpenClash 控制面板密钥 | `VMware1!` |
| 刷新时间（秒） | Dashboard 自动刷新间隔 | `30` |

配置文件 `config.json` 内容如下：

```json
{
  "api_address": "",
  "api_secret": "",
  "api_source_port": "9090",
  "port": "9099",
  "refresh_seconds": 30
}
```

## API 接口

### 内部接口（Dashboard 同源调用，无需认证）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/` | Dashboard 页面 |
| GET | `/api/data` | 获取当前节点数据和统计 |
| POST | `/api/config` | 保存配置 |
| POST | `/api/test` | 测试 OpenClash 连接 |
| POST | `/api/refresh` | 触发 OpenClash 延迟检测 |
| GET | `/api/health` | 健康检查 |

### 外部接口（需携带 Bearer Token）

所有外部接口需要在 Header 中携带：`Authorization: Bearer <API密钥>`

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/summary` | 获取统计摘要 |
| GET | `/api/nodes` | 获取所有节点列表 |
| GET | `/api/regions` | 获取所有地区及节点 |
| GET | `/api/regions/{code}/nodes` | 获取指定地区的节点 |
| GET | `/api/nodes/filter?region=&category=` | 按地区和分类过滤 |
| GET | `/api/nodes/{name}` | 获取指定节点详情 |
| GET | `/api/token` | 获取当前 API 密钥 |

### API 返回示例

```json
// GET /api/summary
{
  "total": 103,
  "fast": 17,
  "normal": 43,
  "high_latency": 17,
  "fault": 26,
  "healthy_count": 60,
  "healthy_pct": 58
}
```

## 节点延迟分类

| 等级 | 延迟范围 | 说明 |
|------|----------|------|
| 🚀 高速 | ≤ 150ms | 优质节点 |
| ✅ 正常 | 151 ~ 240ms | 可用节点 |
| ⚠️ 低延迟 | 241 ~ 500ms | 高延迟节点 |
| ❌ 故障 | > 500ms 或 0ms | 不可用节点 |

## 地区分类

支持以下地区自动识别：香港、新加坡、台湾、日本、美国、英国、德国、土耳其等，未匹配节点归入"其他地区"。

## 镜像说明

| 镜像 | 说明 |
|------|------|
| `ghcr.io/teddyxiong79/site-health-monitor:latest` | 最新稳定版 |
| `ghcr.io/teddyxiong79/site-health-monitor:main` | 主分支最新构建 |

支持平台：`linux/amd64`、`linux/arm64`。

## 构建

```bash
# 本地构建
git clone https://github.com/TeddyXiong79/site-health-monitor.git
cd site-health-monitor
go build -o site-health-monitor .

# Docker 多平台构建
docker buildx build --platform=linux/amd64,linux/arm64 -t ghcr.io/teddyxiong79/site-health-monitor:latest --push .
```

## 版本历史

- **v1.5.1** — 修复并发安全、Bug 优化、Docker 构建完善
- **v1.5.0** — 初始版本

## License

MIT
