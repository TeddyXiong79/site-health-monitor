package main

type Config struct {
    Port           string `json:"port"`
    APIAddress     string `json:"api_address"`
    APISourcePort  string `json:"api_source_port"`
    APISecret      string `json:"api_secret"`
    RefreshSeconds int    `json:"refresh_seconds"`
}

type Node struct {
    Name         string `json:"name"`
    OriginalName string `json:"original_name"`
    Delay        int    `json:"delay"`
    Category     string `json:"category"`  // fast, normal, high_latency, fault
    Region       string `json:"region"`
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
    Version    string
    Stats      Stats
    Regions    []Region
    SafeConfig SafeConfig
    UpdateTime string
}

// SafeConfig 不含敏感信息的配置，供前端渲染使用
type SafeConfig struct {
    APIAddress     string
    APISourcePort  string
    MaskedSecret   string // 脱敏后的密钥，如 "VM***1!"
    RawToken       string // 原始完整 Token，供前端同源请求自动携带 Bearer 认证（服务端渲染注入）
    RefreshSeconds int
}

type RegionConfig struct {
    Code     string
    Name     string
    Patterns []string
}

var RegionConfigs = []RegionConfig{
    {"HK", "中国香港", []string{"-HK-", "BGP-HK", "HK-NF", "HK-"}},
    {"SG", "新加坡", []string{"-SG-", "BGP-SG", "SG-NF", "SG-"}},
    {"TW", "中国台湾", []string{"-TW-", "BGP-TW", "TW-NF", "TW-"}},
    {"JP", "日本", []string{"-JP-", "BGP-JP", "JP-NF", "JP-"}},
    {"US", "美国", []string{"-US-", "BGP-US", "US-NF", "US-"}},
}
