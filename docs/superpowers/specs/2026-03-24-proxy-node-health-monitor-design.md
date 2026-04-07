# 全球代理节点健康监控 - 设计文档

## 1. 项目概述

- **项目名称**：全球代理节点健康监控 (Global Real-time Proxy Node Health Monitor)
- **版本**：v1.4.9
- **技术栈**：Go + HTML/CSS/JS (黑客帝国风格)
- **功能**：从 OpenClash 设备获取节点延迟数据，实时展示健康度状态
- **端口**：9099

## 2. 数据源

- **来源**：运行 OpenClash 的设备页面
- **地址**：用户配置（如 `http://192.168.66.251:9090`）
- **认证**：Bearer Token (`Authorization: Bearer <token>`)
- **数据格式**（OpenClash API 原始字段）：
  - 节点名称字段：`Node_name`
  - 延迟字段：`Latency` (ms)
- **内部字段映射**：
  - `Node_name` → `name`（处理后）
  - `Latency` → `delay`（处理后）
- **首次启动**：不检测网络连通性，等待用户配置数据源

## 3. 配置项

| 配置项 | 默认值 | 说明 |
|--------|--------|------|
| API 地址 | 空 | OpenClash 设备 IP |
| API 端口 | 9090 | OpenClash API 端口 |
| API 密钥 | 空 | Bearer Token |
| 刷新间隔 | 30秒 | 页面自动刷新时间 |

**持久化**：配置保存到 `config.json`，重启后不丢失

## 4. 节点分类逻辑

### 4.1 过滤规则
- 排除节点名含 `COMPATIBLE` 或 `PASS` 的节点
- 显示时去除名字中的 `-BGP-` 字符串

### 4.2 延迟分类

| 分类 | 延迟范围 | 图标 | 颜色 |
|------|----------|------|------|
| 高速节点 | 2ms ≤ delay < 150ms | 🚀 | 蓝色 `#00a8ff` |
| 正常节点 | 150ms ≤ delay < 220ms | ✅ | 绿色 `#00ff88` |
| 低速节点 | 220ms ≤ delay < 350ms | ⚠️ | 黄色 `#ffcc00` |
| 故障节点 | delay = 0 或 > 350ms | ❌ | 红色 `#ff3366` |

> **注意**：边界值 350ms 属于低速节点（<350ms），超过 350ms 才属于故障节点。

### 4.3 地区归类

| 地区 | 关键词 | 图标 |
|------|--------|------|
| 中国香港 | 包含 `-HK-` 或以 `HK` 结尾 | 🇭🇰 |
| 新加坡 | 包含 `-SG-` 或以 `SG` 结尾 | 🇸🇬 |
| 中国台湾 | 包含 `-TW-` 或以 `TW` 结尾 | 🏴TB 旗帜（SVG base64） |
| 日本 | 包含 `-JP-` 或以 `JP` 结尾 | 🇯🇵 |
| 美国 | 包含 `-US-` 或以 `US` 结尾 | 🇺🇸 |
| 其他地区 | 剩余节点 | 🌐 |

> **注意**：使用 `-关键词-` 或 `关键词` 结尾的匹配方式，避免误匹配（如 `PHK` 不会匹配到 `HK`）。

## 5. Web Dashboard 页面结构

### 5.1 第一部分：项目信息
- 标题：全球代理节点健康监控
- 副标题：Global Real-time Proxy Node HealthMonitor
- 版本号：v1.4.9
- 设计者：Design by Teddy

### 5.2 第二部分：健康度汇总
- 4个统计卡片：高速/正常/低速/故障节点数
- 健康度统计：高速+正常节点总数及百分比

### 5.3 第三部分：地区节点列表
- 标题行：节点总数 + 更新时间 + 倒计时
- 折叠卡片式展示各地区节点
- 点击展开显示双列节点详情（名称+延迟）

### 5.4 第四部分：数据源配置
- 输入框：API地址、端口、密钥、刷新间隔
- 按钮：连接测试、配置保存
- 反馈：结果显示3秒后自动消失

## 6. REST API 接口

### 6.1 内部 API（Dashboard 调用）

| 接口 | 方法 | 说明 |
|------|------|------|
| `GET /api/data` | GET | 完整数据（统计+地区+节点） |
| `POST /api/config` | POST | 保存配置 |
| `POST /api/test` | POST | 测试数据源连接 |

### 6.2 外部 API（供其他设备调用）

| 接口 | 方法 | 说明 |
|------|------|------|
| `GET /api/summary` | GET | 汇总统计（健康度等） |
| `GET /api/nodes` | GET | 所有节点详情列表 |
| `GET /api/regions` | GET | 各地区汇总（无详情） |
| `POST /api/token/refresh` | POST | 刷新 API Token |

### 6.3 认证
- **API Token**：程序运行时动态生成，存储在内存中
- 所有外部 API 调用需要 Header：`Authorization: Bearer <token>`
- Token 在 Dashboard 页面显示（设置/Token 区域）
- `/api/token/refresh`：刷新 API Token，新 Token 会返回给调用者
- **内部 API**（Dashboard 调用 `/api/data`, `/api/config`, `/api/test`）：无需认证，通过同源访问控制
- Token 不持久化，重启程序后重新生成

## 7. 响应数据结构

### 7.1 /api/data 完整响应
```json
{
  "stats": {
    "total": 100,
    "fast": 45,
    "normal": 30,
    "high_latency": 15,
    "fault": 10,
    "healthy_count": 75,
    "healthy_pct": 75
  },
  "regions": [
    {
      "name": "中国香港",
      "name_en": "Hong Kong",
      "stats": { "total": 20, "fast": 10, "normal": 8, "high_latency": 2, "fault": 0 },
      "nodes": [
        { "name": "HK-01", "delay": 45, "category": "fast" }
      ]
    }
  ],
  "update_time": "Mar 24 at 09:15"
}
```

### 7.2 /api/summary 响应
```json
{
  "total": 100,
  "fast": 45,
  "normal": 30,
  "high_latency": 15,
  "fault": 10,
  "healthy_count": 75,
  "healthy_pct": 75
}
```

### 7.3 /api/nodes 响应
```json
{
  "nodes": [
    { "name": "HK-01", "delay": 45, "category": "fast", "region": "中国香港" }
  ]
}
```

### 7.4 /api/regions 响应
```json
{
  "regions": [
    { "name": "中国香港", "stats": { "total": 20, "fast": 10, "normal": 8, "high_latency": 2, "fault": 0 } }
  ]
}
```

## 8. 文件结构

```
Site-Health-Monitor/
├── main.go              # 主程序入口
├── config.go           # 配置管理
├── handlers.go         # HTTP 处理器
├── fetcher.go          # OpenClash API 获取
├── processor.go        # 数据处理/分类
├── router.go           # 路由注册
├── templates/
│   └── dashboard.html  # 前端模板（已有）
├── config.json         # 配置文件（运行时生成）
└── docs/
    └── superpowers/
        └── specs/      # 设计文档
```

## 9. 核心依赖

- Go 1.21+
- `github.com/gorilla/mux` - HTTP 路由
- `github.com/spf13/viper` - 配置管理
- 标准库 `net/http` - HTTP 客户端

## 10. 错误处理

- 数据源连接失败：显示错误状态，不阻塞页面渲染
- API 请求超时：5秒超时
- 配置缺失：引导用户进入配置页面
