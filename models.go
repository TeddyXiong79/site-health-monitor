package main

type Config struct {
    Port           string `json:"port"`
    APIAddress     string `json:"api_address"`
    APISourcePort  string `json:"api_source_port"`
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
    Version    string
    Stats      Stats
    Regions    []Region
    Config     Config
    UpdateTime string
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
