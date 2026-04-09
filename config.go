package main

import (
    "log"
    "os"
    "path/filepath"
    "strconv"
    "sync"

    "github.com/spf13/viper"
)

var AppConfig Config
var configMutex sync.RWMutex

// configDir 存放配置文件的目录路径
var configDir string

func GetConfig() Config {
    configMutex.RLock()
    defer configMutex.RUnlock()
    return AppConfig
}

func SetConfig(cfg Config) {
    configMutex.Lock()
    defer configMutex.Unlock()
    AppConfig = cfg
}

// detectConfigDir 检测配置文件存放目录
// 容器内使用 /data/（VOLUME 自动持久化），本地开发使用当前目录
func detectConfigDir() string {
    // 优先检查 /data/ 目录是否存在（容器环境）
    if info, err := os.Stat("/data"); err == nil && info.IsDir() {
        return "/data"
    }
    // 本地开发环境使用当前目录
    return "."
}

func LoadConfig() error {
    configDir = detectConfigDir()
    log.Printf("[配置] 配置目录: %s", configDir)

    viper.SetConfigName("config")
    viper.SetConfigType("json")
    viper.AddConfigPath(configDir)

    // 设置默认值
    viper.SetDefault("port", "9099")
    viper.SetDefault("api_source_port", "9090")
    viper.SetDefault("refresh_seconds", 120)
    if err := viper.ReadInConfig(); err != nil {
        if _, ok := err.(viper.ConfigFileNotFoundError); ok {
            // 配置文件不存在，使用默认配置
            cfg := Config{
                Port:           "9099",
                APISourcePort:  "9090",
                RefreshSeconds: 120, // 与 viper.SetDefault 一致
            }
            SetConfig(cfg)
            // 自动创建默认配置文件
            viper.Set("port", cfg.Port)
            viper.Set("api_source_port", cfg.APISourcePort)
            viper.Set("refresh_seconds", cfg.RefreshSeconds)
            _ = viper.SafeWriteConfigAs(filepath.Join(configDir, "config.json"))
            return nil
        }
        return err
    }

    cfg := Config{
        Port:           viper.GetString("port"),
        APIAddress:     viper.GetString("api_address"),
        APISourcePort:  viper.GetString("api_source_port"),
        APISecret:      viper.GetString("api_secret"),
        RefreshSeconds: viper.GetInt("refresh_seconds"),
    }
    SetConfig(cfg)

    return nil
}

func SaveConfig(cfg Config) error {
    if err := ValidateConfig(cfg); err != nil {
        log.Printf("[配置保存] 校验失败: %v", err)
        return err
    }

    viper.Set("port", cfg.Port)
    viper.Set("api_address", cfg.APIAddress)
    viper.Set("api_source_port", cfg.APISourcePort)
    viper.Set("api_secret", cfg.APISecret)
    viper.Set("refresh_seconds", cfg.RefreshSeconds)

    configPath := filepath.Join(configDir, "config.json")

    if _, err := os.Stat(configPath); os.IsNotExist(err) {
        if err := viper.SafeWriteConfigAs(configPath); err != nil {
            log.Printf("[配置保存] 创建配置文件失败: %v", err)
            return err
        }
        log.Printf("[配置保存] 成功（新文件）: address=%s, port=%s, secret=***", cfg.APIAddress, cfg.APISourcePort)
    } else {
        if err := viper.WriteConfigAs(configPath); err != nil {
            log.Printf("[配置保存] 写入配置文件失败: %v", err)
            return err
        }
        log.Printf("[配置保存] 成功: address=%s, port=%s, secret=***", cfg.APIAddress, cfg.APISourcePort)
    }

    SetConfig(cfg)
    return nil
}

func ValidateConfig(cfg Config) error {
    if cfg.APISourcePort == "" {
        return &ConfigError{Field: "api_source_port", Message: "数据源端口不能为空"}
    }
    port, err := strconv.Atoi(cfg.APISourcePort)
    if err != nil || port < 1 || port > 65535 {
        return &ConfigError{Field: "api_source_port", Message: "数据源端口必须是 1-65535 之间的数字"}
    }
    if cfg.APISecret == "" {
        return &ConfigError{Field: "api_secret", Message: "API密钥不能为空"}
    }
    if cfg.RefreshSeconds < 10 || cfg.RefreshSeconds > 3600 {
        return &ConfigError{Field: "refresh_seconds", Message: "刷新间隔必须在 10-3600 秒之间"}
    }
    return nil
}

type ConfigError struct {
    Field   string
    Message string
}

func (e *ConfigError) Error() string {
    return e.Message
}
