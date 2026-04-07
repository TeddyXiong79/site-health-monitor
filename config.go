package main

import (
    "log"
    "os"
    "sync"

    "github.com/spf13/viper"
)

var AppConfig Config
var configMutex sync.RWMutex

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

func LoadConfig() error {
    viper.SetConfigName("config")
    viper.SetConfigType("json")
    viper.AddConfigPath(".")

    // 设置默认值
    viper.SetDefault("port", "9099")
    viper.SetDefault("api_source_port", "9090")
    viper.SetDefault("refresh_seconds", 30)
    if err := viper.ReadInConfig(); err != nil {
        if _, ok := err.(viper.ConfigFileNotFoundError); ok {
            // 配置文件不存在，使用默认配置
            SetConfig(Config{
                Port:           "9099",
                APISourcePort:  "9090",
                RefreshSeconds: 30,
            })
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

    // SafeWriteConfig 只在文件不存在时成功，WriteConfig 只在文件存在时成功
    if _, err := os.Stat("config.json"); os.IsNotExist(err) {
        if err := viper.SafeWriteConfig(); err != nil {
            log.Printf("[配置保存] 创建配置文件失败: %v", err)
            return err
        }
        log.Printf("[配置保存] 成功（新文件）: address=%s, port=%s, secret=***", cfg.APIAddress, cfg.APISourcePort)
    } else {
        if err := viper.WriteConfig(); err != nil {
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
    if cfg.APISecret == "" {
        return &ConfigError{Field: "api_secret", Message: "API密钥不能为空"}
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
