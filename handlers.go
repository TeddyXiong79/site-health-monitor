package main

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

var dashboardTemplate *template.Template

const appVersion = "v1.6.1"

func init() {
	parseTemplates()
}

func parseTemplates() {
	funcs := template.FuncMap{
		"add":        func(a, b int) int { return a + b },
		"percentage": func(total, part int) int {
			if total == 0 {
				return 0
			}
			return (part * 100) / total
		},
		"escapeAttr": func(s string) string {
			s = strings.ReplaceAll(s, "&", "&amp;")
			s = strings.ReplaceAll(s, "\"", "&quot;")
			s = strings.ReplaceAll(s, "<", "&lt;")
			s = strings.ReplaceAll(s, ">", "&gt;")
			s = strings.ReplaceAll(s, "`", "&#96;")
			s = strings.ReplaceAll(s, "$", "&#36;")
			return s
		},
		"regionFlag": func(name string) string {
			flags := map[string]string{
				"中国香港": "🇭🇰",
				"新加坡":   "🇸🇬",
				"中国台湾": "TW",
				"日本":    "🇯🇵",
				"美国":    "🇺🇸",
				"其他地区": "🌏",
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
		"maskToken": maskToken,
	}
	dashboardTemplate = template.New("dashboard.html").Funcs(funcs)
	dashboardTemplate = template.Must(dashboardTemplate.ParseFiles("templates/dashboard.html"))
}

func RenderDashboard(w http.ResponseWriter, r *http.Request) {
	cfg := GetConfig()
	nodes, err := FetchNodesCached(cfg)
	if err != nil {
		log.Printf("FetchNodes error: %v", err)
		nodes = []OpenClashNode{}
	}

	processedNodes, stats := ProcessNodes(nodes)
	regions := GroupByRegion(processedNodes)

	now := time.Now()
	updateTime := now.Format("Jan 02 at 15:04")

	safeConfig := SafeConfig{
		APIAddress:     cfg.APIAddress,
		APISourcePort:  cfg.APISourcePort,
		MaskedSecret:   maskToken(cfg.APISecret),
		RefreshSeconds: cfg.RefreshSeconds,
	}

	data := DashboardData{
		Version:    appVersion,
		Stats:      stats,
		Regions:    regions,
		SafeConfig: safeConfig,
		UpdateTime: updateTime,
	}

	if err := dashboardTemplate.Execute(w, map[string]interface{}{
		"Data": data,
	}); err != nil {
		log.Printf("模板渲染失败: %v", err)
	}
}

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

func APIGetData(w http.ResponseWriter, r *http.Request) {
	// 内部 API，无需认证（Dashboard 同源调用）

	nodes, err := FetchNodesCached(GetConfig())
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":       err.Error(),
			"stats":       Stats{},
			"regions":     []Region{},
			"update_time": time.Now().Format("Jan 02 at 15:04"),
		})
		return
	}

	processedNodes, stats := ProcessNodes(nodes)
	regions := GroupByRegion(processedNodes)

	now := time.Now()
	updateTime := now.Format("Jan 02 at 15:04")

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
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

	nodes, err := FetchNodesCached(GetConfig())
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(APIResponse{Success: false, Message: err.Error()})
		return
	}
	_, stats := ProcessNodes(nodes)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	json.NewEncoder(w).Encode(stats)
}

func APIGetNodes(w http.ResponseWriter, r *http.Request) {
	if !validateToken(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	nodes, err := FetchNodesCached(GetConfig())
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(APIResponse{Success: false, Message: err.Error()})
		return
	}
	processedNodes, _ := ProcessNodes(nodes)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"nodes": processedNodes,
	})
}

func APIGetRegions(w http.ResponseWriter, r *http.Request) {
	if !validateToken(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	nodes, err := FetchNodesCached(GetConfig())
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(APIResponse{Success: false, Message: err.Error()})
		return
	}
	processedNodes, _ := ProcessNodes(nodes)
	regions := GroupByRegion(processedNodes)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"regions": regions,
	})
}

func APIHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

func APIRefresh(w http.ResponseWriter, r *http.Request) {
	if err := TriggerDelayCheck(GetConfig()); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(APIResponse{Success: false, Message: err.Error()})
		return
	}

	InvalidateCache()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{Success: true, Message: "延迟检查已触发，请稍后刷新获取最新数据"})
}

func APIGetRegionNodes(w http.ResponseWriter, r *http.Request) {
	if !validateToken(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	regionCode := vars["region"]
	regionName := regionCodeToName(regionCode)

	nodes, err := FetchNodesCached(GetConfig())
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(APIResponse{Success: false, Message: err.Error()})
		return
	}
	processedNodes, _ := ProcessNodes(nodes)
	regions := GroupByRegion(processedNodes)

	for _, region := range regions {
		if region.Name == regionName {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
			w.Header().Set("Pragma", "no-cache")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"nodes": region.Nodes,
				"stats": region.Stats,
			})
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"nodes": []Node{},
		"stats": RegionStats{},
	})
}

func APIGetNodeFilter(w http.ResponseWriter, r *http.Request) {
	if !validateToken(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	regionCode := r.URL.Query().Get("region")
	category := r.URL.Query().Get("category")

	nodes, err := FetchNodesCached(GetConfig())
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(APIResponse{Success: false, Message: err.Error()})
		return
	}
	processedNodes, _ := ProcessNodes(nodes)

	filtered := make([]Node, 0)
	for _, n := range processedNodes {
		if regionCode != "" && regionCodeToName(strings.ToUpper(regionCode)) != n.Region {
			continue
		}
		if category != "" && n.Category != category {
			continue
		}
		filtered = append(filtered, n)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"nodes": filtered,
	})
}

func APIGetNodeDetail(w http.ResponseWriter, r *http.Request) {
	if !validateToken(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	nodeName := vars["name"]

	nodes, err := FetchNodesCached(GetConfig())
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(APIResponse{Success: false, Message: err.Error()})
		return
	}
	processedNodes, _ := ProcessNodes(nodes)

	for _, n := range processedNodes {
		if n.Name == nodeName {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
			w.Header().Set("Pragma", "no-cache")
			json.NewEncoder(w).Encode(n)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": "node not found",
	})
}

func APISaveConfig(w http.ResponseWriter, r *http.Request) {
	// 已有密钥时要求 Bearer Token 认证（首次配置允许无认证）
	if GetConfig().APISecret != "" && !validateToken(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var cfg Config
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIResponse{Success: false, Message: "无效的请求"})
		return
	}

	// 保留服务端口，不允许用户修改
	cfg.Port = GetConfig().Port

	// 密钥留空时保留原密钥（前端 placeholder 承诺"留空则不修改"）
	if cfg.APISecret == "" {
		cfg.APISecret = GetConfig().APISecret
	}

	if err := SaveConfig(cfg); err != nil {
		json.NewEncoder(w).Encode(APIResponse{Success: false, Message: err.Error()})
		return
	}

	InvalidateCache()
	json.NewEncoder(w).Encode(APIResponse{Success: true, Message: "配置已保存"})
}

func APITestConnection(w http.ResponseWriter, r *http.Request) {
	// 已有密钥时要求 Bearer Token 认证（首次配置允许无认证）
	if GetConfig().APISecret != "" && !validateToken(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var cfg Config
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIResponse{Success: false, Message: "无效的请求"})
		return
	}

	success, message := TestConnection(cfg)
	json.NewEncoder(w).Encode(APIResponse{Success: success, Message: message})
}

func validateToken(r *http.Request) bool {
	cfg := GetConfig()
	if cfg.APISecret == "" {
		return false
	}
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return false
	}
	token := strings.TrimPrefix(auth, "Bearer ")
	return subtle.ConstantTimeCompare([]byte(token), []byte(cfg.APISecret)) == 1
}

// APISwitchProxy 设置 🔰国外流量 策略组的当前代理
func APISwitchProxy(w http.ResponseWriter, r *http.Request) {
	var req struct {
		NodeName string `json:"node_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.NodeName == "" {
		http.Error(w, "node_name is required", http.StatusBadRequest)
		return
	}

	if err := SwitchProxy(GetConfig(), req.NodeName); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(APIResponse{Success: false, Message: err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{Success: true, Message: fmt.Sprintf("已将 🔰国外流量 切换至 %s", req.NodeName)})
}

// regionCodeToName 转换区域简码到完整名称
func regionCodeToName(code string) string {
	for _, cfg := range RegionConfigs {
		if cfg.Code == code {
			return cfg.Name
		}
	}
	if code == "Other" {
		return "其他地区"
	}
	return code
}

// maskToken 对密钥进行脱敏处理
func maskToken(token string) string {
	if token == "" {
		return ""
	}
	if len(token) <= 4 {
		return "***"
	}
	return token[:2] + "***" + token[len(token)-2:]
}
