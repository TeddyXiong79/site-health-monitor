# 全球代理节点健康监控 - 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 构建一个 Go Web 程序，从 OpenClash 设备获取代理节点延迟数据，通过 Dashboard 页面展示健康度状态，并提供 REST API 接口供外部调用。

**Architecture:**
- Go + gorilla/mux 路由
- Viper 配置管理
- Go template 渲染 Dashboard HTML
- Bearer Token 认证
- 配置持久化到 config.json

**Tech Stack:** Go 1.21+, github.com/gorilla/mux, github.com/spf13/viper

---

## 文件结构

```
Site-Health-Monitor/
├── main.go              # 程序入口，端口 9099
├── config.go            # 配置管理：读取/保存 config.json
├── models.go            # 数据结构定义
├── handlers.go          # HTTP 处理器（Dashboard + API）
├── fetcher.go           # OpenClash API 获取
├── processor.go         # 数据处理：过滤、分类、延迟计算
├── router.go            # 路由注册
├── templates/
│   └── dashboard.html   # 前端模板（已有）
├── config.json         # 配置文件（运行时生成）
└── docs/
    └── superpowers/
        ├── specs/      # 设计文档
        └── plans/      # 实现计划
```

---

## Task 1: 配置管理与数据模型

**Files:**
- Create: `config.go`
- Create: `models.go`
- Create: `config.json` (空配置)

- [ ] **Step 1: 创建 models.go - 数据结构定义**

```go
package main

type Config struct {
    APIAddress     string `json:"api_address"`
    APIPort        string `json:"api_port"`
    APISecret      string `json:"api_secret"`
    RefreshSeconds int    `json:"refresh_seconds"`
}

type Node struct {
    Name      string `json:"name"`
    Delay     int    `json:"delay"`
    Category  string `json:"category"`  // fast, normal, high_latency, fault
    Region    string `json:"region"`
}

type RegionStats struct {
    Total       int `json:"total"`
    Fast       int `json:"fast"`
    Normal     int `json:"normal"`
    HighLatency int `json:"high_latency"`
    Fault      int `json:"fault"`
}

type Region struct {
    Name    string      `json:"name"`
    NameEN  string      `json:"name_en"`
    Stats   RegionStats `json:"stats"`
    Nodes   []Node     `json:"nodes"`
}

type Stats struct {
    Total        int `json:"total"`
    Fast        int `json:"fast"`
    Normal      int `json:"normal"`
    HighLatency int `json:"high_latency"`
    Fault       int `json:"fault"`
    HealthyCount int `json:"healthy_count"`
    HealthyPct  int `json:"healthy_pct"`
}

type DashboardData struct {
    Version  string
    Stats    Stats
    Regions  []Region
    Config   Config
    APIToken string
}
```

- [ ] **Step 2: 创建 config.go - 配置管理**

```go
package main

import (
    "github.com/spf13/viper"
)

var AppConfig Config

func LoadConfig() error {
    viper.SetConfigName("config")
    viper.SetConfigType("json")
    viper.AddConfigPath(".")

    // 设置默认值
    viper.SetDefault("api_port", "9090")
    viper.SetDefault("refresh_seconds", 30)

    if err := viper.ReadInConfig(); err != nil {
        if _, ok := err.(viper.ConfigFileNotFoundError); ok {
            // 配置文件不存在，使用默认配置
            AppConfig = Config{
                APIPort:        "9090",
                RefreshSeconds: 30,
            }
            return nil
        }
        return err
    }

    return viper.Unmarshal(&AppConfig)
}

func SaveConfig(cfg Config) error {
    viper.Set("api_address", cfg.APIAddress)
    viper.Set("api_port", cfg.APIPort)
    viper.Set("api_secret", cfg.APISecret)
    viper.Set("refresh_seconds", cfg.RefreshSeconds)
    return viper.WriteConfig()
}
```

- [ ] **Step 3: 创建空 config.json**

```json
{
    "api_address": "",
    "api_port": "9090",
    "api_secret": "",
    "refresh_seconds": 30
}
```

- [ ] **Step 4: 提交**

```bash
git add config.go models.go config.json
git commit -m "feat: add config management and data models"
```

---

## Task 2: OpenClash 数据获取

**Files:**
- Create: `fetcher.go`

- [ ] **Step 1: 创建 fetcher.go**

```go
package main

import (
    "encoding/json"
    "fmt"
    "net/http"
    "time"
)

type OpenClashResponse struct {
    Nodes []OpenClashNode `json:"proxies"`
}

type OpenClashNode struct {
    NodeName string `json:"Node_name"`
    Latency  int    `json:"Latency"`
}

func FetchNodes(cfg Config) ([]OpenClashNode, error) {
    if cfg.APIAddress == "" {
        return []OpenClashNode{}, nil
    }

    url := fmt.Sprintf("http://%s:%s/proxies", cfg.APIAddress, cfg.APIPort)

    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return nil, err
    }

    if cfg.APISecret != "" {
        req.Header.Set("Authorization", "Bearer "+cfg.APISecret)
    }

    client := &http.Client{
        Timeout: 5 * time.Second,
    }

    resp, err := client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var result OpenClashResponse
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }

    return result.Nodes, nil
}

func TestConnection(cfg Config) (bool, string) {
    if cfg.APIAddress == "" {
        return false, "API地址未配置"
    }

    nodes, err := FetchNodes(cfg)
    if err != nil {
        return false, fmt.Sprintf("连接失败: %v", err)
    }

    return true, fmt.Sprintf("连接成功，获取到 %d 个节点", len(nodes))
}
```

- [ ] **Step 2: 提交**

```bash
git add fetcher.go
git commit -m "feat: add OpenClash data fetcher"
```

---

## Task 3: 数据处理与分类

**Files:**
- Create: `processor.go`

- [ ] **Step 1: 创建 processor.go - 过滤和分类逻辑**

```go
package main

import (
    "strings"
)

func ProcessNodes(rawNodes []OpenClashNode) ([]Node, Stats) {
    var nodes []Node
    stats := Stats{}

    for _, n := range rawNodes {
        // 过滤规则：排除 COMPATIBLE 和 PASS
        name := n.NodeName
        if strings.Contains(name, "COMPATIBLE") || strings.Contains(name, "PASS") {
            continue
        }

        // 分类延迟
        category := categorizeDelay(n.Latency)

        // 地区归类
        region := categorizeRegion(name)

        // 去除 -BGP- 字符串
        displayName := strings.ReplaceAll(name, "-BGP-", "")

        nodes = append(nodes, Node{
            Name:     displayName,
            Delay:    n.Latency,
            Category: category,
            Region:   region,
        })
    }

    // 计算统计
    for _, n := range nodes {
        stats.Total++
        switch n.Category {
        case "fast":
            stats.Fast++
        case "normal":
            stats.Normal++
        case "high_latency":
            stats.HighLatency++
        case "fault":
            stats.Fault++
        }
    }

    stats.HealthyCount = stats.Fast + stats.Normal
    if stats.Total > 0 {
        stats.HealthyPct = (stats.HealthyCount * 100) / stats.Total
    }

    return nodes, stats
}

func categorizeDelay(latency int) string {
    if latency == 0 || latency > 350 {
        return "fault"
    }
    if latency < 150 {
        return "fast"
    }
    if latency < 220 {
        return "normal"
    }
    return "high_latency"
}

func categorizeRegion(name string) string {
    // 匹配规则：-XX- 或 XX 结尾
    upperName := strings.ToUpper(name)

    if strings.Contains(upperName, "-HK-") || strings.HasSuffix(upperName, "-HK") || strings.HasSuffix(upperName, "HK") {
        return "中国香港"
    }
    if strings.Contains(upperName, "-SG-") || strings.HasSuffix(upperName, "-SG") || strings.HasSuffix(upperName, "SG") {
        return "新加坡"
    }
    if strings.Contains(upperName, "-TW-") || strings.HasSuffix(upperName, "-TW") || strings.HasSuffix(upperName, "TW") {
        return "中国台湾"
    }
    if strings.Contains(upperName, "-JP-") || strings.HasSuffix(upperName, "-JP") || strings.HasSuffix(upperName, "JP") {
        return "日本"
    }
    if strings.Contains(upperName, "-US-") || strings.HasSuffix(upperName, "-US") || strings.HasSuffix(upperName, "US") {
        return "美国"
    }
    return "其他地区"
}

func GroupByRegion(nodes []Node) []Region {
    regionMap := make(map[string][]Node)

    for _, n := range nodes {
        regionMap[n.Region] = append(regionMap[n.Region], n)
    }

    var regions []Region
    regionNames := []string{"中国香港", "新加坡", "中国台湾", "日本", "美国", "其他地区"}
    regionEnglish := map[string]string{
        "中国香港": "Hong Kong",
        "新加坡":   "Singapore",
        "中国台湾": "Taiwan",
        "日本":    "Japan",
        "美国":    "United States",
        "其他地区": "Other",
    }

    for _, name := range regionNames {
        if nodeList, ok := regionMap[name]; ok {
            stats := RegionStats{}
            for _, n := range nodeList {
                stats.Total++
                switch n.Category {
                case "fast":
                    stats.Fast++
                case "normal":
                    stats.Normal++
                case "high_latency":
                    stats.HighLatency++
                case "fault":
                    stats.Fault++
                }
            }
            regions = append(regions, Region{
                Name:    name,
                NameEN:  regionEnglish[name],
                Stats:   stats,
                Nodes:   nodeList,
            })
        }
    }

    return regions
}
```

- [ ] **Step 2: 提交**

```bash
git add processor.go
git commit -m "feat: add node processing and categorization logic"
```

---

## Task 4: HTTP 处理器

**Files:**
- Create: `handlers.go`

- [ ] **Step 1: 创建 handlers.go - HTTP 处理器**

```go
package main

import (
    "crypto/rand"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "html/template"
    "net/http"
    "strings"
    "time"

    "github.com/gorilla/mux"
)

var apiToken string

func init() {
    generateAPIToken()
}

func generateAPIToken() {
    b := make([]byte, 16)
    rand.Read(b)
    apiToken = hex.EncodeToString(b)
}

var dashboardTemplate = template.Must(template.ParseFiles("templates/dashboard.html"))

func RenderDashboard(w http.ResponseWriter, r *http.Request) {
    nodes, err := FetchNodes(AppConfig)
    if err != nil {
        nodes = []OpenClashNode{}
    }

    processedNodes, stats := ProcessNodes(nodes)
    regions := GroupByRegion(processedNodes)

    now := time.Now()
    updateTime := now.Format("Jan 02 at 15:04")

    data := DashboardData{
        Version:  "v1.4.9",
        Stats:    stats,
        Regions:  regions,
        Config:   AppConfig,
        APIToken: apiToken,
    }

    // 添加模板函数
    funcs := template.FuncMap{
        "add":        func(a, b int) int { return a + b },
        "percentage": func(total, part int) int {
            if total == 0 {
                return 0
            }
            return (part * 100) / total
        },
        "currentTime": func() string {
            return updateTime
        },
        "regionFlag": func(name string) string {
            flags := map[string]string{
                "中国香港": "🇭🇰",
                "新加坡":   "🇸🇬",
                "中国台湾": "TW",
                "日本":    "🇯🇵",
                "美国":    "🇺🇸",
                "其他地区": "🌐",
            }
            return flags[name]
        },
        "regionEnglish": func(name string) string {
            en := map[string]string{
                "中国香港": "Hong Kong",
                "新加坡":   "Singapore",
                "中国台湾": "Taiwan",
                "日本":    "Japan",
                "美国":    "United States",
                "其他地区": "Other",
            }
            return en[name]
        },
    }

    t := template.New("dashboard.html").Funcs(funcs)
    t = template.Must(t.ParseFiles("templates/dashboard.html"))

    t.Execute(w, map[string]interface{}{
        "Data": data,
    })
}

type APIResponse struct {
    Success bool        `json:"success"`
    Message string      `json:"message,omitempty"`
    Data    interface{} `json:"data,omitempty"`
}

func APIGetData(w http.ResponseWriter, r *http.Request) {
    if !validateToken(r) {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    nodes, _ := FetchNodes(AppConfig)
    processedNodes, stats := ProcessNodes(nodes)
    regions := GroupByRegion(processedNodes)

    now := time.Now()
    updateTime := now.Format("Jan 02 at 15:04")

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "stats":       stats,
        "regions":     regions,
        "update_time": updateTime,
    })
}

func APIGetSummary(w http.ResponseWriter, r *http.Request) {
    if !validateToken(r) {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    nodes, _ := FetchNodes(AppConfig)
    _, stats := ProcessNodes(nodes)

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(stats)
}

func APIGetNodes(w http.ResponseWriter, r *http.Request) {
    if !validateToken(r) {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    nodes, _ := FetchNodes(AppConfig)
    processedNodes, _ := ProcessNodes(nodes)

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "nodes": processedNodes,
    })
}

func APIGetRegions(w http.ResponseWriter, r *http.Request) {
    if !validateToken(r) {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    nodes, _ := FetchNodes(AppConfig)
    processedNodes, _ := ProcessNodes(nodes)
    regions := GroupByRegion(processedNodes)

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "regions": regions,
    })
}

func APIRefreshToken(w http.ResponseWriter, r *http.Request) {
    if !validateToken(r) {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    generateAPIToken()

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "success": true,
        "token":   apiToken,
    })
}

func APISaveConfig(w http.ResponseWriter, r *http.Request) {
    var cfg Config
    if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
        json.NewEncoder(w).Encode(APIResponse{Success: false, Message: "无效的请求"})
        return
    }

    if err := SaveConfig(cfg); err != nil {
        json.NewEncoder(w).Encode(APIResponse{Success: false, Message: err.Error()})
        return
    }

    AppConfig = cfg
    json.NewEncoder(w).Encode(APIResponse{Success: true})
}

func APITestConnection(w http.ResponseWriter, r *http.Request) {
    var cfg Config
    if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
        json.NewEncoder(w).Encode(APIResponse{Success: false, Message: "无效的请求"})
        return
    }

    success, message := TestConnection(cfg)
    json.NewEncoder(w).Encode(APIResponse{Success: success, Message: message})
}

func validateToken(r *http.Request) bool {
    auth := r.Header.Get("Authorization")
    if auth == "" {
        return false
    }
    token := strings.TrimPrefix(auth, "Bearer ")
    return token == apiToken
}
```

- [ ] **Step 2: 提交**

```bash
git add handlers.go
git commit -m "feat: add HTTP handlers for dashboard and API"
```

---

## Task 5: 路由与主程序

**Files:**
- Create: `router.go`
- Create: `main.go`

- [ ] **Step 1: 创建 router.go**

```go
package main

import (
    "net/http"

    "github.com/gorilla/mux"
)

func SetupRouter() *mux.Router {
    r := mux.NewRouter()

    // Dashboard
    r.HandleFunc("/", RenderDashboard).Methods("GET")

    // 内部 API（Dashboard 调用，无需认证）
    r.HandleFunc("/api/data", APIGetData).Methods("GET")
    r.HandleFunc("/api/config", APISaveConfig).Methods("POST")
    r.HandleFunc("/api/test", APITestConnection).Methods("POST")

    // 外部 API（需要 Bearer Token 认证）
    r.HandleFunc("/api/summary", APIGetSummary).Methods("GET")
    r.HandleFunc("/api/nodes", APIGetNodes).Methods("GET")
    r.HandleFunc("/api/regions", APIGetRegions).Methods("GET")
    r.HandleFunc("/api/token/refresh", APIRefreshToken).Methods("POST")

    // 静态文件
    r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.Dir("static")))

    return r
}
```

- [ ] **Step 2: 创建 main.go**

```go
package main

import (
    "log"
    "net/http"
)

func main() {
    // 加载配置
    if err := LoadConfig(); err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    // 设置路由
    router := SetupRouter()

    // 启动服务器
    log.Println("服务器启动，端口: 9099")
    log.Println("Dashboard: http://localhost:9099/")
    log.Fatal(http.ListenAndServe(":9099", router))
}
```

- [ ] **Step 3: 提交**

```bash
git add router.go main.go
git commit -m "feat: add router and main entry point"
```

---

## Task 6: 模板适配

**Files:**
- Modify: `templates/dashboard.html` (从 Dashboard.html 复制)

- [ ] **Step 1: 复制 Dashboard.html 到 templates 目录**

```bash
cp Dashboard.html templates/dashboard.html
```

- [ ] **Step 2: 检查并修正模板语法**

Dashboard.html 使用了 Go template 语法 `{{.Data.xxx}}`，需要确保与 handlers.go 中的数据结构匹配。

主要需要确认：
- `{{.Data.Version}}` → DashboardData.Version ✓
- `{{.Data.Stats.Fast}}` → DashboardData.Stats.Fast ✓
- 地区循环 `{{range .Data.Regions}}` → DashboardData.Regions ✓

- [ ] **Step 3: 提交**

```bash
git add templates/dashboard.html
git commit -m "feat: add dashboard template"
```

---

## Task 7: 编译测试

**Files:**
- Test: 全局编译测试

- [ ] **Step 1: 初始化 Go 模块**

```bash
go mod init site-health-monitor
go mod tidy
```

- [ ] **Step 2: 编译测试**

```bash
go build -o site-health-monitor .
```

预期：无错误，生成可执行文件

- [ ] **Step 3: 测试运行**

```bash
./site-health-monitor &
curl http://localhost:9099/
```

预期：返回 Dashboard HTML

- [ ] **Step 4: 测试 API Token 认证**

```bash
# 无 Token
curl http://localhost:9099/api/summary
预期: 401 Unauthorized

# 带 Token（页面刷新获取新 Token）
curl -H "Authorization: Bearer <token>" http://localhost:9099/api/summary
预期: 返回 JSON 汇总数据
```

- [ ] **Step 5: 提交**

```bash
git add go.mod
git commit -m "chore: add go module"
```

---

## API 接口文档

### 内部 API（Dashboard 同源调用，无需认证）

| 接口 | 方法 | 说明 | 请求体 |
|------|------|------|--------|
| `/api/data` | GET | 完整数据 | - |
| `/api/config` | POST | 保存配置 | `{"api_address":"", "api_port":"9090", "api_secret":"", "refresh_seconds":30}` |
| `/api/test` | POST | 测试连接 | `{"api_address":"", "api_port":"9090", "api_secret":""}` |

### 外部 API（需要 Bearer Token 认证）

| 接口 | 方法 | 说明 | Header |
|------|------|------|--------|
| `/api/summary` | GET | 汇总统计 | `Authorization: Bearer <token>` |
| `/api/nodes` | GET | 节点详情 | `Authorization: Bearer <token>` |
| `/api/regions` | GET | 地区汇总 | `Authorization: Bearer <token>` |
| `/api/token/refresh` | POST | 刷新 Token | `Authorization: Bearer <token>` |

### Python 调用示例

```python
import requests

# 获取汇总
token = "页面显示的token"
headers = {"Authorization": f"Bearer {token}"}
r = requests.get("http://localhost:9099/api/summary", headers=headers)
print(r.json())

# 获取所有节点
r = requests.get("http://localhost:9099/api/nodes", headers=headers)
print(r.json())
```

### Curl 调用示例

```bash
# 获取汇总
curl -H "Authorization: Bearer YOUR_TOKEN" http://localhost:9099/api/summary

# 获取节点列表
curl -H "Authorization: Bearer YOUR_TOKEN" http://localhost:9099/api/nodes
```
