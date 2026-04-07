package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var httpClient = &http.Client{
	Timeout: 5 * time.Second,
}

var delayCheckClient = &http.Client{
	Timeout: 30 * time.Second,
}

// fetchProxies 执行 HTTP GET 并解析 OpenClash proxies 响应，供 FetchNodes 和 TestConnection 共用
func fetchProxies(cfg Config) (*OpenClashResponse, error) {
	if cfg.APIAddress == "" {
		return nil, errors.New("API地址未配置")
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
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result OpenClashResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
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
			// 尝试从 extra 中获取延迟（部分节点格式），取最新的
			for _, extra := range proxy.Extra {
				if len(extra.History) > 0 {
					delay = extra.History[len(extra.History)-1].Delay
					break
				}
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
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
