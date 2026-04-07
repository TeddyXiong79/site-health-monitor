# 全球代理节点健康监控 - API 接口文档

> **版本**: v1.4.9
> **更新时间**: 2026-03-24

---

## 功能概述

本系统提供两大核心能力：

1. **节点延迟数据读取** - 从 OpenClash 获取代理节点的延迟数据
2. **节点延迟主动刷新** - 触发 OpenClash 对所有节点进行实时延迟测试

**刷新流程**: 触发延迟测试 → OpenClash 同步测试所有节点 → 返回最新延迟数据

---

## 认证方式

外部 API 接口使用 **Bearer Token** 认证。

**Token 值**: 数据源配置中的 `api_secret`（对外API调用密钥）

**请求头**:
```
Authorization: Bearer <token>
```

---

## 接口列表

### 1. 健康检查

**接口**: `GET /api/health`
**认证**: 无需认证
**用途**: 负载均衡器/监控平台探测服务状态

**响应示例**:
```json
{
  "status": "ok",
  "timestamp": "2026-03-24T11:19:44+08:00"
}
```

**curl 示例**:
```bash
curl http://localhost:9099/api/health
```

---

### 2. 获取 Dashboard 数据（内部接口）

**接口**: `GET /api/data`
**认证**: 无需认证（供 Dashboard 页面 AJAX 调用）
**用途**: 获取完整的 Dashboard 数据（统计、地区分组、更新时间等）

**响应示例**:
```json
{
  "stats": {
    "total": 103,
    "fast": 15,
    "normal": 33,
    "high_latency": 24,
    "fault": 31,
    "healthy_count": 48,
    "healthy_pct": 47
  },
  "regions": [
    {
      "name": "中国香港",
      "name_en": "Hong Kong",
      "stats": {
        "total": 36,
        "fast": 5,
        "normal": 20,
        "high_latency": 8,
        "fault": 3
      },
      "nodes": [...]
    }
  ],
  "update_time": "Mar 24 at 22:10"
}
```

**说明**: 此接口在页面加载或刷新时，会先调用 `/api/refresh` 触发 OpenClash 延迟测试后再获取数据。

**curl 示例**:
```bash
curl http://localhost:9099/api/data
```

---

### 3. 触发节点延迟刷新

**接口**: `POST /api/refresh`
**认证**: 无需认证
**用途**: 触发 OpenClash 对所有节点进行实时延迟测试（同步等待完成）

**请求参数**: 无（使用配置中的数据源地址、端口、密钥和组名）

**配置依赖**:
- `api_address` - 数据源地址
- `api_source_port` - 数据源端口
- `api_secret` - 数据源密钥
- `group_name` - OpenClash 中的代理组名称（如 `🔰国外流量`）

**响应示例**:
```json
{
  "success": true,
  "message": "延迟检查已完成"
}
```

**失败响应**:
```json
{
  "success": false,
  "message": "连接失败: context deadline exceeded"
}
```

**curl 示例**:
```bash
curl -X POST http://localhost:9099/api/refresh
```

**完整刷新流程示例**:
```bash
# 1. 触发延迟测试（等待完成）
curl -X POST http://localhost:9099/api/refresh

# 2. 获取最新数据
curl http://localhost:9099/api/data
```

---

### 4. 测试数据源连接

**接口**: `POST /api/test`
**认证**: 无需认证
**用途**: 测试 OpenClash 数据源连接是否正常

**请求体**:
```json
{
  "api_address": "192.168.66.251",
  "api_source_port": "9090",
  "api_secret": "VMware1!"
}
```

**响应示例**:
```json
{
  "success": true,
  "message": "连接成功，共130个节点"
}
```

**curl 示例**:
```bash
curl -X POST http://localhost:9099/api/test \
  -H "Content-Type: application/json" \
  -d '{"api_address":"192.168.66.251","api_source_port":"9090","api_secret":"VMware1!"}'
```

---

### 5. 保存配置

**接口**: `POST /api/config`
**认证**: 无需认证
**用途**: 保存数据源配置（修改后需要重启刷新才能生效）

**请求体**:
```json
{
  "api_address": "192.168.66.251",
  "api_source_port": "9090",
  "api_secret": "VMware1!",
  "refresh_seconds": 30,
  "group_name": "🔰国外流量"
}
```

**响应示例**:
```json
{
  "success": true,
  "message": "配置已保存"
}
```

**curl 示例**:
```bash
curl -X POST http://localhost:9099/api/config \
  -H "Content-Type: application/json" \
  -d '{"api_address":"192.168.66.251","api_source_port":"9090","api_secret":"VMware1!","refresh_seconds":30,"group_name":"🔰国外流量"}'
```

---

### 6. 获取汇总统计

**接口**: `GET /api/summary`
**认证**: Bearer Token
**用途**: 获取节点健康度汇总

**响应示例**:
```json
{
  "total": 103,
  "fast": 15,
  "normal": 33,
  "high_latency": 24,
  "fault": 31,
  "healthy_count": 48,
  "healthy_pct": 47
}
```

**字段说明**:
| 字段 | 类型 | 说明 |
|------|------|------|
| `total` | int | 节点总数 |
| `fast` | int | 高速节点 (<150ms) |
| `normal` | int | 正常节点 (150-219ms) |
| `high_latency` | int | 低速节点 (220-350ms) |
| `fault` | int | 故障节点 (>350ms 或 0ms) |
| `healthy_count` | int | 健康节点数 (fast + normal) |
| `healthy_pct` | int | 健康百分比 |

**curl 示例**:
```bash
curl -H "Authorization: Bearer VMware1!" http://localhost:9099/api/summary
```

---

### 7. 获取所有节点详情

**接口**: `GET /api/nodes`
**认证**: Bearer Token
**用途**: 获取全部代理节点列表及详情

**响应示例**:
```json
{
  "nodes": [
    {
      "name": "🇭🇰 V410U-1X-HK-NF",
      "delay": 173,
      "category": "normal",
      "region": "中国香港"
    },
    {
      "name": "🇭🇰 V308U-1X-HK-NF",
      "delay": 177,
      "category": "normal",
      "region": "中国香港"
    }
  ]
}
```

**字段说明**:
| 字段 | 类型 | 说明 |
|------|------|------|
| `name` | string | 节点名称（已格式化，含 emoji 标识） |
| `delay` | int | 延迟值 (ms)，0 表示故障 |
| `category` | string | 分类: `fast` / `normal` / `high_latency` / `fault` |
| `region` | string | 地区: `中国香港` / `新加坡` / `中国台湾` / `日本` / `美国` / `其他地区` |

**curl 示例**:
```bash
curl -H "Authorization: Bearer VMware1!" http://localhost:9099/api/nodes
```

---

### 8. 获取按地区分组的节点

**接口**: `GET /api/regions`
**认证**: Bearer Token
**用途**: 获取按地区分组的节点统计及列表

**响应示例**:
```json
{
  "regions": [
    {
      "name": "中国香港",
      "name_en": "Hong Kong",
      "stats": {
        "total": 36,
        "fast": 5,
        "normal": 20,
        "high_latency": 8,
        "fault": 3
      },
      "nodes": [
        {
          "name": "🇭🇰 V410U-1X-HK-NF",
          "delay": 173,
          "category": "normal",
          "region": "中国香港"
        }
      ]
    },
    {
      "name": "新加坡",
      "name_en": "Singapore",
      "stats": {
        "total": 19,
        "fast": 3,
        "normal": 10,
        "high_latency": 4,
        "fault": 2
      },
      "nodes": [...]
    }
  ]
}
```

**curl 示例**:
```bash
curl -H "Authorization: Bearer VMware1!" http://localhost:9099/api/regions
```

---

### 9. 获取指定地区节点

**接口**: `GET /api/regions/{region}/nodes`
**认证**: Bearer Token
**用途**: 获取指定地区的节点列表及统计

**URL 参数**:
| 值 | 说明 |
|------|------|
| `HK` | 中国香港 |
| `SG` | 新加坡 |
| `TW` | 中国台湾 |
| `JP` | 日本 |
| `US` | 美国 |
| `Other` | 其他地区 |

**响应示例**:
```json
{
  "nodes": [
    {
      "name": "🇭🇰 V410U-1X-HK-NF",
      "delay": 173,
      "category": "normal",
      "region": "中国香港"
    }
  ],
  "stats": {
    "total": 36,
    "fast": 5,
    "normal": 20,
    "high_latency": 8,
    "fault": 3
  }
}
```

**curl 示例**:
```bash
curl -H "Authorization: Bearer VMware1!" http://localhost:9099/api/regions/HK/nodes
```

---

### 10. 按条件过滤节点

**接口**: `GET /api/nodes/filter`
**认证**: Bearer Token
**用途**: 按地区或类别过滤节点

**URL 参数**:
| 参数 | 值 | 说明 |
|------|------|------|
| `region` | `HK` / `SG` / `TW` / `JP` / `US` / `Other` | 地区筛选 |
| `category` | `fast` / `normal` / `high_latency` / `fault` | 类别筛选 |

**响应示例**:
```json
{
  "nodes": [
    {
      "name": "🇭🇰 V410U-1X-HK-NF",
      "delay": 173,
      "category": "normal",
      "region": "中国香港"
    }
  ]
}
```

**curl 示例**:
```bash
# 过滤香港的正常节点
curl -H "Authorization: Bearer VMware1!" \
  "http://localhost:9099/api/nodes/filter?region=HK&category=normal"

# 仅按地区过滤
curl -H "Authorization: Bearer VMware1!" \
  "http://localhost:9099/api/nodes/filter?region=HK"

# 仅按类别过滤
curl -H "Authorization: Bearer VMware1!" \
  "http://localhost:9099/api/nodes/filter?category=fast"
```

---

### 11. 获取单个节点详情

**接口**: `GET /api/nodes/{name}`
**认证**: Bearer Token
**用途**: 获取指定节点的详细信息

**URL 参数**:
- `name`: 节点名称（需 URL 编码）

**响应示例**:
```json
{
  "name": "应急节点",
  "delay": 490,
  "category": "fault",
  "region": "其他地区"
}
```

**节点不存在时**:
```json
{
  "error": "node not found"
}
```

**curl 示例**:
```bash
# 中文节点名需 URL 编码
curl -H "Authorization: Bearer VMware1!" \
  "http://localhost:9099/api/nodes/%E5%BA%94%E6%80%A5%E8%8A%82%E7%82%B9"
```

---

### 12. 获取当前 API 密钥

**接口**: `GET /api/token`
**认证**: Bearer Token
**用途**: 获取当前配置的 API 密钥

**响应示例**:
```json
{
  "token": "VMware1!"
}
```

**curl 示例**:
```bash
curl -H "Authorization: Bearer VMware1!" http://localhost:9099/api/token
```

---

## Python 调用示例

```python
import urllib.request
import urllib.parse
import json

BASE_URL = "http://localhost:9099"
TOKEN = "VMware1!"


def api_get(path, params=None):
    """GET 请求封装（带认证）"""
    url = f"{BASE_URL}{path}"
    if params:
        url += "?" + urllib.parse.urlencode(params)

    req = urllib.request.Request(
        url,
        headers={"Authorization": f"Bearer {TOKEN}"}
    )
    with urllib.request.urlopen(req) as resp:
        return json.loads(resp.read().decode())


def api_get_no_auth(path):
    """无需认证的 GET 请求"""
    req = urllib.request.Request(f"{BASE_URL}{path}")
    with urllib.request.urlopen(req) as resp:
        return json.loads(resp.read().decode())


def api_post_no_auth(path, data):
    """POST 请求（无需认证）"""
    req = urllib.request.Request(
        f"{BASE_URL}{path}",
        data=json.dumps(data).encode(),
        headers={"Content-Type": "application/json"}
    )
    with urllib.request.urlopen(req) as resp:
        return json.loads(resp.read().decode())


# === 使用示例 ===

# 1. 健康检查
health = api_get_no_auth("/api/health")
print(f"状态: {health['status']}")

# 2. 触发延迟刷新（重要！获取最新数据前必须调用）
refresh = api_post_no_auth("/api/refresh", {})
if refresh.get("success"):
    print("延迟刷新成功")
else:
    print(f"刷新失败: {refresh.get('message')}")

# 3. 获取 Dashboard 数据
data = api_get_no_auth("/api/data")
print(f"总节点: {data['stats']['total']}, 健康率: {data['stats']['healthy_pct']}%")

# 4. 测试连接
test = api_post_no_auth("/api/test", {
    "api_address": "192.168.66.251",
    "api_source_port": "9090",
    "api_secret": "VMware1!"
})
print(test.get("message"))

# 5. 保存配置
config = api_post_no_auth("/api/config", {
    "api_address": "192.168.66.251",
    "api_source_port": "9090",
    "api_secret": "VMware1!",
    "refresh_seconds": 30,
    "group_name": "🔰国外流量"
})
print(config.get("message"))

# 6. 获取汇总
summary = api_get("/api/summary")
print(f"总节点: {summary['total']}, 健康率: {summary['healthy_pct']}%")

# 7. 获取所有节点
nodes = api_get("/api/nodes")
print(f"节点数: {len(nodes['nodes'])}")

# 8. 获取地区分组
regions = api_get("/api/regions")
for r in regions['regions']:
    print(f"{r['name']}: {r['stats']['total']}节点")

# 9. 获取指定地区节点
hk_nodes = api_get("/api/regions/HK/nodes")
print(f"香港节点: {len(hk_nodes['nodes'])}个")

# 10. 过滤节点
normal_hk = api_get("/api/nodes/filter", {"region": "HK", "category": "normal"})
print(f"香港正常节点: {len(normal_hk['nodes'])}个")

# 11. 获取单个节点
node = api_get("/api/nodes/%E5%BA%94%E6%80%A5%E8%8A%82%E7%82%B9")
print(f"节点: {node['name']}, 延迟: {node['delay']}ms")

# 12. 获取 Token
token = api_get("/api/token")
print(f"Token: {token['token']}")
```

---

## 错误响应

**未授权 (401)**:
```
Unauthorized
```

**请求错误 (400)**:
```json
{
  "success": false,
  "message": "无效的请求"
}
```

**刷新失败**:
```json
{
  "success": false,
  "message": "连接失败: context deadline exceeded"
}
```

---

## 节点分类标准

| 类别 | 延迟范围 | 说明 |
|------|----------|------|
| `fast` | < 150ms | 高速节点 |
| `normal` | 150-219ms | 正常节点 |
| `high_latency` | 220-350ms | 低速节点 |
| `fault` | > 350ms 或 0ms | 故障节点 |

---

## 地区代码对照

| 代码 | 地区名 | 英文名 |
|------|--------|--------|
| `HK` | 中国香港 | Hong Kong |
| `SG` | 新加坡 | Singapore |
| `TW` | 中国台湾 | Taiwan |
| `JP` | 日本 | Japan |
| `US` | 美国 | United States |
| `Other` | 其他地区 | Other |

---

## 配置说明

### 配置文件: `config.json`

```json
{
  "port": "9099",
  "api_address": "192.168.66.251",
  "api_source_port": "9090",
  "api_secret": "VMware1!",
  "refresh_seconds": 30,
  "group_name": "🔰国外流量"
}
```

| 字段 | 说明 | 默认值 |
|------|------|--------|
| `port` | 服务端口 | 9099 |
| `api_address` | OpenClash 数据源地址 | - |
| `api_source_port` | OpenClash 数据源端口 | 9090 |
| `api_secret` | OpenClash 访问密钥 | - |
| `refresh_seconds` | 页面自动刷新间隔（秒） | 30 |
| `group_name` | OpenClash 代理组名称（用于触发延迟测试） | 🔰国外流量 |

---

## 注意事项

1. **延迟刷新**: `/api/refresh` 会同步等待 OpenClash 完成所有节点测试，超时时间 30 秒
2. **URL 编码**: 节点名称含特殊字符（如 emoji、中文）时需 URL 编码
3. **Bearer Token**: 大小写敏感，需与 `config.json` 中的 `api_secret` 完全一致
4. **时区**: 所有时间戳使用 RFC3339 格式（本地时区）
5. **数据更新**: 建议在获取数据前先调用 `/api/refresh` 确保获取最新延迟
