package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// maxResponseSize 限制上游响应体最大为 10MB，防止 OOM
const maxResponseSize = 10 * 1024 * 1024

var httpClient = &http.Client{
	Timeout: 5 * time.Second,
}

var delayCheckClient = &http.Client{
	Timeout: 30 * time.Second,
}

var switchProxyClient = &http.Client{
	Timeout: 3 * time.Second,
}

// fetchProxies 执行 HTTP GET 并解析 OpenClash proxies 响应，供 FetchNodes 和 TestConnection 共用
func fetchProxies(cfg Config) (*OpenClashResponse, error) {
	if cfg.APIAddress == "" {
		return nil, errors.New("API地址未配置")
	}

	// SSRF 防护：验证地址合法性
	if err := validateAPIAddress(cfg.APIAddress); err != nil {
		return nil, err
	}

	url := fmt.Sprintf("http://%s:%s/proxies", cfg.APIAddress, cfg.APISourcePort)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	if cfg.APISecret != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.APISecret)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result OpenClashResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseSize)).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &result, nil
}

type OpenClashNode struct {
	NodeName string `json:"name"`
	Latency  int    `json:"delay"`
}

type OpenClashResponse struct {
	Proxies map[string]OpenClashProxy `json:"proxies"`
}

type OpenClashProxy struct {
	Name    string                   `json:"name"`
	Type    string                   `json:"type"`
	Alive   bool                     `json:"alive"`
	History []HistoryEntry           `json:"history"`
	Extra   map[string]ExtraEntry   `json:"extra"`
}

type HistoryEntry struct {
	Time  string `json:"time"`
	Delay int    `json:"delay"`
}

type ExtraEntry struct {
	Alive   bool           `json:"alive"`
	History []HistoryEntry `json:"history"`
}

func FetchNodes(cfg Config) ([]OpenClashNode, error) {
	result, err := fetchProxies(cfg)
	if err != nil {
		return []OpenClashNode{}, err
	}

	nodes := make([]OpenClashNode, 0, len(result.Proxies))
	for name, proxy := range result.Proxies {
		// 跳过 Selector、Direct、Reject 等非代理节点类型
		if proxy.Type == "Selector" || proxy.Type == "Direct" || proxy.Type == "Reject" ||
			proxy.Type == "RejectDrop" || proxy.Type == "Compatible" || proxy.Type == "Pass" {
			continue
		}

		// 获取延迟：从 history 中获取最新的延迟值（最后一个元素）
		delay := 0
		if len(proxy.History) > 0 {
			delay = proxy.History[len(proxy.History)-1].Delay
		} else if proxy.Extra != nil {
			// 尝试从 extra 中获取延迟（部分节点格式）
			// Go map 遍历顺序不确定，取 history 最长的条目（包含最新数据）
			bestDelay := 0
			maxHistoryLen := 0
			for _, extra := range proxy.Extra {
				if len(extra.History) > maxHistoryLen {
					maxHistoryLen = len(extra.History)
					bestDelay = extra.History[len(extra.History)-1].Delay
				}
			}
			if maxHistoryLen > 0 {
				delay = bestDelay
			}
		}

		// 如果 alive 为 false，延迟保持历史值不变
		// alive 状态已由 processor.go 中的 categorizeDelay 处理（延迟为0视为故障）

		nodes = append(nodes, OpenClashNode{
			NodeName: name,
			Latency:  delay,
		})
	}

	return nodes, nil
}

func TestConnection(cfg Config) (bool, string) {
	result, err := fetchProxies(cfg)
	if err != nil {
		return false, classifyConnectError(err)
	}

	if len(result.Proxies) == 0 {
		return false, "未获取到任何代理节点"
	}

	return true, fmt.Sprintf("连接成功，共%d个节点", len(result.Proxies))
}

// classifyConnectError 根据网络错误返回友好的中文提示
func classifyConnectError(err error) string {
	if err == nil {
		return "连接失败: 未知错误"
	}
	errStr := err.Error()

	// 关键词匹配不同错误类型
	switch {
	// 连接被拒绝
	case containsAny(errStr, "connection refused", "ECONNREFUSED"):
		return "连接被拒绝，请检查OpenClash管理面板端口是否正确"
	// IP/主机不可达
	case containsAny(errStr, "no such host", "lookup fail", "ENOTFOUND", "server misbehaving", "i/o timeout", "timeout exceeded"):
		return "IP地址错误或主机不可达，请检查数据源地址是否正确"
	// 连接超时
	case containsAny(errStr, "timeout", "TIMEOUT", "deadline exceeded", "Client.Timeout"):
		return "连接超时，服务器响应过慢，请稍后重试"
	// 连接重置
	case containsAny(errStr, "connection reset", "ECONNRESET"):
		return "连接被重置，请检查OpenClash是否正常运行"
	// 网络不可达
	case containsAny(errStr, "network is unreachable", "ENETUNREACH", "no route to host"):
		return "网络不可达，请检查服务器网络连接"
	// 地址不可用
	case containsAny(errStr, "address not available", "EADDRNOTAVAIL"):
		return "IP地址不可用，请检查数据源地址是否正确"
	// 默认
	default:
		return fmt.Sprintf("连接失败: %v", err)
	}
}

func containsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

func TriggerDelayCheck(cfg Config) error {
	if cfg.APIAddress == "" {
		return fmt.Errorf("API地址未配置")
	}
	if err := validateAPIAddress(cfg.APIAddress); err != nil {
		return err
	}
	groupName := "🔰国外流量"

	encodedGroup := url.PathEscape(groupName)
	testURL := url.QueryEscape("http://www.gstatic.com/generate_204")
	urlStr := fmt.Sprintf("http://%s:%s/group/%s/delay?url=%s&timeout=5000",
		cfg.APIAddress, cfg.APISourcePort, encodedGroup, testURL)

	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	if cfg.APISecret != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.APISecret)
	}

	resp, err := delayCheckClient.Do(req)
	if err != nil {
		return errors.New(classifyConnectError(err))
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return fmt.Errorf("密钥错误，请检查API密钥是否正确")
	}
	if resp.StatusCode == 404 {
		return fmt.Errorf("分组名称不存在，请检查OpenClash配置")
	}
	if resp.StatusCode == 503 {
		return fmt.Errorf("服务不可用，OpenClash可能未运行")
	}
	if resp.StatusCode >= 500 {
		return fmt.Errorf("OpenClash服务器错误 (%d)", resp.StatusCode)
	}
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// SwitchProxy 设置 🔰国外流量 策略组的当前选中代理
func SwitchProxy(cfg Config, nodeName string) error {
	if cfg.APIAddress == "" {
		return fmt.Errorf("API地址未配置")
	}
	if err := validateAPIAddress(cfg.APIAddress); err != nil {
		return err
	}
	groupName := "🔰国外流量"

	encodedGroup := url.PathEscape(groupName)
	urlStr := fmt.Sprintf("http://%s:%s/proxies/%s", cfg.APIAddress, cfg.APISourcePort, encodedGroup)

	body, _ := json.Marshal(map[string]string{"name": nodeName})
	req, err := http.NewRequest("PUT", urlStr, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.APISecret != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.APISecret)
	}

	resp, err := switchProxyClient.Do(req)
	if err != nil {
		return errors.New(classifyConnectError(err))
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return fmt.Errorf("密钥错误，请检查API密钥是否正确")
	}
	if resp.StatusCode == 404 {
		return fmt.Errorf("分组或节点不存在，请检查名称是否正确")
	}
	if resp.StatusCode >= 500 {
		return fmt.Errorf("OpenClash服务器错误 (%d)", resp.StatusCode)
	}
	if resp.StatusCode != 200 && resp.StatusCode != 204 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// validateAPIAddress 验证 API 地址合法性，防止 SSRF 攻击
func validateAPIAddress(addr string) error {
	if addr == "" {
		return fmt.Errorf("API地址不能为空")
	}
	// 禁止包含 URL scheme 或路径
	if strings.Contains(addr, "://") || strings.Contains(addr, "/") {
		return fmt.Errorf("API地址格式错误：不能包含协议或路径")
	}
	// 解析为 IP 并检查是否为回环/不可路由地址
	ip := net.ParseIP(addr)
	if ip != nil {
		if ip.IsLoopback() || ip.IsUnspecified() {
			return fmt.Errorf("不允许使用回环或不可路由地址")
		}
	}
	return nil
}
