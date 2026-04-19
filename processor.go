package main

import (
	"sort"
	"strings"
)

// 包级变量，避免每次调用都创建 Replacer
var hyphenReplacer = strings.NewReplacer(
	"1XHK", "1X-HK",
	"2XHK", "2X-HK",
	"4XHK", "4X-HK",
	"1XSG", "1X-SG",
	"2XSG", "2X-SG",
	"4XSG", "4X-SG",
	"1XTW", "1X-TW",
	"2XTW", "2X-TW",
	"4XTW", "4X-TW",
	"1XJP", "1X-JP",
	"2XJP", "2X-JP",
	"4XJP", "4X-JP",
	"1XUS", "1X-US",
	"2XUS", "2X-US",
	"4XUS", "4X-US",
	"1XUK", "1X-UK",
	"2XUK", "2X-UK",
	"4XUK", "4X-UK",
	"1XDE", "1X-DE",
	"2XDE", "2X-DE",
	"4XDE", "4X-DE",
	"1XTR", "1X-TR",
	"2XTR", "2X-TR",
	"4XTR", "4X-TR",
)

func ProcessNodes(rawNodes []OpenClashNode) ([]Node, Stats) {
	var nodes []Node
	stats := Stats{}

	// 单次遍历：同时处理节点和更新统计
	for _, n := range rawNodes {
		name := n.NodeName
		category := categorizeDelay(n.Latency)
		region := categorizeRegion(name)

		// 处理显示名称
		displayName := strings.ReplaceAll(name, "-BGP-", "")
		displayName = hyphenReplacer.Replace(displayName)

		nodes = append(nodes, Node{
			Name:         displayName,
			OriginalName: name,
			Delay:        n.Latency,
			Category:     category,
			Region:       region,
		})

		// 更新统计
		stats.Total++
		switch category {
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
	if latency <= 0 || latency > 500 {
		return "fault"
	}
	if latency >= 1 && latency <= 150 {
		return "fast"
	}
	if latency <= 240 {
		return "normal"
	}
	return "high_latency"
}

// categoryPriority 返回健康等级的排序优先级（数字越小越健康）
func categoryPriority(category string) int {
	switch category {
	case "fast":
		return 0
	case "normal":
		return 1
	case "high_latency":
		return 2
	case "fault":
		return 3
	default:
		return 4
	}
}

func categorizeRegion(name string) string {
	for _, cfg := range RegionConfigs {
		for _, pattern := range cfg.Patterns {
			if strings.Contains(strings.ToUpper(name), strings.ToUpper(pattern)) {
				return cfg.Name
			}
		}
	}
	return "其他地区"
}

func GroupByRegion(nodes []Node) []Region {
	regionMap := make(map[string][]Node, 6)

	for _, n := range nodes {
		regionMap[n.Region] = append(regionMap[n.Region], n)
	}

	regions := make([]Region, 0, 6)
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
			// 排序规则：先按健康等级（fast < normal < high_latency < fault），再按延迟升序
			// 修复：不再用 Delay==0 作为故障标记（负延迟和 0 都是故障，但大于 240 的高延迟也是问题）
			sort.Slice(nodeList, func(i, j int) bool {
				pi := categoryPriority(nodeList[i].Category)
				pj := categoryPriority(nodeList[j].Category)
				if pi != pj {
					return pi < pj
				}
				// 同一健康等级内：故障节点延迟可能为负/0，不具备比较意义，保持稳定顺序
				if nodeList[i].Category == "fault" {
					return false
				}
				return nodeList[i].Delay < nodeList[j].Delay
			})

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
