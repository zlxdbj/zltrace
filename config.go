package zltrace

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// ============================================================================
// TraceConfig - 追踪系统配置
// ============================================================================

// TraceConfig 追踪系统配置
type TraceConfig struct {
	// 是否启用追踪（总开关）
	Enabled bool `mapstructure:"enabled"`

	// 服务名称（用于追踪系统中标识本服务）
	ServiceName string `mapstructure:"service_name"`

	// 采样配置
	Sampler SamplerConfig `mapstructure:"sampler"`

	// Exporter 配置
	Exporter ExporterConfig `mapstructure:"exporter"`

	// 批量处理配置
	Batch BatchConfig `mapstructure:"batch"`
}

// SamplerConfig 采样器配置
type SamplerConfig struct {
	Type  string  `mapstructure:"type"`
	Ratio float64 `mapstructure:"ratio"`
}

// ExporterConfig Exporter 配置
type ExporterConfig struct {
	Type       string            `mapstructure:"type"`
	OTLP       OTLPConfig        `mapstructure:"otlp"`
	MaxQueueSize int               `mapstructure:"max_queue_size"`
}

// OTLPConfig OTLP gRPC 配置
type OTLPConfig struct {
	Endpoint string        `mapstructure:"endpoint"`
	Timeout  int           `mapstructure:"timeout"`
	Insecure bool          `mapstructure:"insecure"`
}

// BatchConfig 批量处理配置
type BatchConfig struct {
	BatchSize    int `mapstructure:"batch_size"`
	Timeout      int `mapstructure:"timeout"`
	MaxQueueSize int `mapstructure:"max_queue_size"`
}

// ============================================================================
// 配置加载
// ============================================================================

// LoadConfig 从配置文件加载配置
// 支持从多个位置读取配置文件：
//   1. ./zltrace.yaml (当前目录)
//   2. $ZLTRACE_CONFIG 环境变量指定的路径
//   3. /etc/zltrace/config.yaml (系统配置目录)
//   4. 如果都没有，返回默认配置
func LoadConfig() (*TraceConfig, error) {
	// 1. 创建默认配置
	config := &TraceConfig{
		Enabled:     true,
		ServiceName: detectServiceName(),
		Sampler: SamplerConfig{
			Type:  "always_on",
			Ratio: 1.0,
		},
		Exporter: ExporterConfig{
			Type: "stdout", // 默认输出到日志（降级模式）
			OTLP: OTLPConfig{
				Endpoint: "localhost:4317",
				Timeout:  10,
				Insecure: true,
			},
			MaxQueueSize: 2048,
		},
		Batch: BatchConfig{
			BatchSize:    512,
			Timeout:      5,
			MaxQueueSize: 2048,
		},
	}

	// 2. 尝试从配置文件读取
	configPaths := []string{
		"./zltrace.yaml",
		os.Getenv("ZLTRACE_CONFIG"),
		"/etc/zltrace/config.yaml",
	}

	// 尝试从多个路径读取配置文件
	for _, path := range configPaths {
		if path == "" {
			continue
		}

		if _, err := os.Stat(path); err == nil {
			// 文件存在，尝试加载
			viper.SetConfigFile(path)
			if err := viper.ReadInConfig(); err == nil {
				if viper.IsSet("trace") {
					if err := viper.UnmarshalKey("trace", config); err != nil {
						return nil, fmt.Errorf("failed to unmarshal trace config from %s: %w", path, err)
					}
				}
				break // 成功加载配置，退出循环
			}
		}
	}

	// 3. 尝试从 Viper 中读取（兼容旧配置方式）
	if viper.IsSet("trace") {
		if err := viper.UnmarshalKey("trace", config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal trace config: %w", err)
		}
	}

	// 3. 设置默认值
	if config.ServiceName == "" {
		config.ServiceName = detectServiceName()
	}
	if config.Sampler.Type == "" {
		config.Sampler.Type = "always_on"
	}
	if config.Exporter.Type == "" {
		config.Exporter.Type = "stdout" // 默认降级模式
	}
	if config.Batch.BatchSize == 0 {
		config.Batch.BatchSize = 512
	}
	if config.Batch.Timeout == 0 {
		config.Batch.Timeout = 5
	}
	if config.Batch.MaxQueueSize == 0 {
		config.Batch.MaxQueueSize = 2048
	}
	if config.Exporter.OTLP.Endpoint == "" {
		config.Exporter.OTLP.Endpoint = "localhost:4317"
	}
	if config.Exporter.OTLP.Timeout == 0 {
		config.Exporter.OTLP.Timeout = 10
	}

	// 4. 验证配置
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid trace config: %w", err)
	}

	return config, nil
}

// detectServiceName 自动检测服务名称
// 优先级: 环境变量 > 配置文件 > 可执行文件名 > 默认值
func detectServiceName() string {
	// 方式1: 从环境变量读取
	if name := os.Getenv("SERVICE_NAME"); name != "" {
		return name
	}
	if name := os.Getenv("APP_NAME"); name != "" {
		return name
	}

	// 方式2: 从配置文件读取
	if viper.IsSet("app.name") {
		return viper.GetString("app.name")
	}

	// 方式3: 使用默认值
	return "zltrace"
}

// validateConfig 验证配置
func validateConfig(config *TraceConfig) error {
	// 如果未启用追踪，直接返回
	if !config.Enabled {
		return nil
	}

	// 验证 exporter 类型
	switch config.Exporter.Type {
	case "otlp", "stdout", "none":
		// 有效值
	default:
		return fmt.Errorf("invalid exporter type: %s (must be otlp, stdout, or none)", config.Exporter.Type)
	}

	// 如果是 otlp，验证必要配置
	if config.Exporter.Type == "otlp" && config.Exporter.OTLP.Endpoint == "" {
		return fmt.Errorf("trace.exporter.otlp.endpoint is required when exporter type is otlp")
	}

	return nil
}

// ============================================================================
// 配置示例（用于文档）
// ============================================================================

// GetExampleConfig 返回配置示例（用于注释或文档）
func GetExampleConfig() string {
	return `
# ========== 追踪系统配置 ==========
trace:
  # 是否启用追踪（总开关）
  enabled: true

  # 服务名称（用于追踪系统中标识本服务）
  service_name: go_shield

  # 采样配置
  sampler:
    # 采样类型: always_on, never, traceid_ratio, parent_based
    type: always_on
    # 采样比率（0.0 - 1.0），仅当 type=traceid_ratio 时生效
    ratio: 1.0

  # Exporter 配置（决定追踪数据发送到哪里）
  exporter:
    # 导出类型: otlp, stdout, none
    # - otlp: 发送到追踪系统（SkyWalking、Jaeger 等）
    # - stdout: 输出到日志（降级模式，不发送到追踪系统）
    # - none: 不发送追踪数据（只生成 trace_id）
    type: stdout

    # OTLP gRPC 配置（type=otlp 时生效）
    otlp:
      # SkyWalking OAP 服务地址
      endpoint: localhost:4317
      # 连接超时时间（秒）
      timeout: 10
      # 是否使用 insecure 连接（开发环境）
      insecure: true

  # 批量处理配置
  batch:
    # 批量发送的最大 span 数量
    batch_size: 512
    # 批量发送的超时时间（秒）
    timeout: 5
    # 最大队列大小
    max_queue_size: 2048
`
}
