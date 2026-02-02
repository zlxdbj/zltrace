package zltrace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

// LoadConfig 从配置文件加载配置（向后兼容函数）
// 使用默认的 ConfigLoader 加载配置
func LoadConfig() (*TraceConfig, error) {
	loader := NewConfigLoader()
	return loader.Load()
}

// ============================================================================
// ConfigLoader - 灵活的配置加载器（参考 zllog 设计）
// ============================================================================

// ConfigLoader 配置加载器
// 支持多种配置来源，按优先级查找：
//   1. trace.yaml（独立配置文件）
//   2. application.yaml（项目配置文件）
//   3. application_{ENV}.yaml（环境配置）
//   4. zltrace.yaml（向后兼容）
//   5. $ZLTRACE_CONFIG 环境变量指定路径
//   6. /etc/zltrace/config.yaml（系统配置目录）
//   7. 默认配置
type ConfigLoader struct {
	// 配置文件查找目录（默认为当前目录和 resource）
	configDirs []string
	// 环境名称（dev/test/prod）
	envName string
}

// NewConfigLoader 创建配置加载器
func NewConfigLoader() *ConfigLoader {
	return &ConfigLoader{
		configDirs: []string{".", "resource"}, // 默认从当前目录和 resource 目录查找
		envName:    detectEnv(),
	}
}

// SetConfigDirs 设置配置文件查找目录
func (l *ConfigLoader) SetConfigDirs(dirs ...string) {
	l.configDirs = dirs
}

// SetEnv 设置环境名称
func (l *ConfigLoader) SetEnv(env string) {
	l.envName = env
}

// Load 加载配置
// 按优先级查找配置文件，如果都找不到则使用默认配置
func (l *ConfigLoader) Load() (*TraceConfig, error) {
	// 1. 尝试从 trace.yaml 加载（独立配置文件，推荐）
	if config, err := l.loadFromTraceYAML(); config != nil && err == nil {
		return config, nil
	}

	// 2. 尝试从 application.yaml 加载
	if config, err := l.loadFromAppYAML("application.yaml"); config != nil && err == nil {
		return config, nil
	}

	// 3. 尝试从 application_{ENV}.yaml 加载
	if l.envName != "" {
		appEnvFile := fmt.Sprintf("application_%s.yaml", l.envName)
		if config, err := l.loadFromAppYAML(appEnvFile); config != nil && err == nil {
			return config, nil
		}
	}

	// 4. 尝试从 zltrace.yaml 加载（向后兼容）
	for _, dir := range l.configDirs {
		configPath := filepath.Join(dir, "zltrace.yaml")
		if config, err := l.loadFromTraceYAMLFile(configPath); config != nil && err == nil {
			return config, nil
		}
	}

	// 5. 尝试从 $ZLTRACE_CONFIG 环境变量加载
	if envPath := os.Getenv("ZLTRACE_CONFIG"); envPath != "" {
		if config, err := l.loadFromTraceYAMLFile(envPath); config != nil && err == nil {
			return config, nil
		}
	}

	// 6. 尝试从 /etc/zltrace/config.yaml 加载
	if config, err := l.loadFromTraceYAMLFile("/etc/zltrace/config.yaml"); config != nil && err == nil {
		return config, nil
	}

	// 7. 使用默认配置
	return l.getDefaultConfig(), nil
}

// loadFromTraceYAML 从独立的 trace.yaml 加载配置
func (l *ConfigLoader) loadFromTraceYAML() (*TraceConfig, error) {
	for _, dir := range l.configDirs {
		configPath := filepath.Join(dir, "trace.yaml")
		config, err := l.loadFromTraceYAMLFile(configPath)
		if err != nil {
			return nil, err
		}
		if config != nil {
			return config, nil
		}
	}
	return nil, nil
}

// loadFromTraceYAMLFile 从指定的 trace.yaml 文件加载配置
func (l *ConfigLoader) loadFromTraceYAMLFile(configPath string) (*TraceConfig, error) {
	// 检查文件是否存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, nil
	}

	// 使用 viper 加载
	v := viper.New()
	v.SetConfigFile(configPath)

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	return l.parseTraceConfig(v)
}

// loadFromAppYAML 从 application.yaml 加载 trace 配置
func (l *ConfigLoader) loadFromAppYAML(filename string) (*TraceConfig, error) {
	for _, dir := range l.configDirs {
		configPath := filepath.Join(dir, filename)

		// 检查文件是否存在
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			continue
		}

		// 使用 viper 加载
		v := viper.New()
		v.SetConfigFile(configPath)

		if err := v.ReadInConfig(); err != nil {
			continue // 文件读取失败，尝试下一个
		}

		// 检查是否有 trace 配置项
		if !v.IsSet("trace") {
			continue
		}

		config, err := l.parseAppTraceConfig(v)
		if err != nil {
			return nil, err
		}
		if config != nil {
			return config, nil
		}
	}

	return nil, nil
}

// parseTraceConfig 解析 trace.yaml 配置（直接格式）
// trace.yaml 格式：
//   service_name: my_service
//   enabled: true
//   exporter:
//     type: otlp
//     otlp:
//       endpoint: localhost:4317
func (l *ConfigLoader) parseTraceConfig(v *viper.Viper) (*TraceConfig, error) {
	serviceName := detectServiceName()
	if v.IsSet("service_name") {
		serviceName = v.GetString("service_name")
	}

	config := l.getDefaultConfig()
	config.ServiceName = serviceName

	// 覆盖配置
	if v.IsSet("enabled") {
		config.Enabled = v.GetBool("enabled")
	}
	if v.IsSet("sampler.type") {
		config.Sampler.Type = v.GetString("sampler.type")
	}
	if v.IsSet("sampler.ratio") {
		config.Sampler.Ratio = v.GetFloat64("sampler.ratio")
	}
	if v.IsSet("exporter.type") {
		config.Exporter.Type = v.GetString("exporter.type")
	}
	if v.IsSet("exporter.otlp.endpoint") {
		config.Exporter.OTLP.Endpoint = v.GetString("exporter.otlp.endpoint")
	}
	if v.IsSet("exporter.otlp.timeout") {
		config.Exporter.OTLP.Timeout = v.GetInt("exporter.otlp.timeout")
	}
	if v.IsSet("exporter.otlp.insecure") {
		config.Exporter.OTLP.Insecure = v.GetBool("exporter.otlp.insecure")
	}
	if v.IsSet("exporter.max_queue_size") {
		config.Exporter.MaxQueueSize = v.GetInt("exporter.max_queue_size")
	}
	if v.IsSet("batch.batch_size") {
		config.Batch.BatchSize = v.GetInt("batch.batch_size")
	}
	if v.IsSet("batch.timeout") {
		config.Batch.Timeout = v.GetInt("batch.timeout")
	}
	if v.IsSet("batch.max_queue_size") {
		config.Batch.MaxQueueSize = v.GetInt("batch.max_queue_size")
	}

	// 验证配置
	if err := validateConfig(config); err != nil {
		return nil, err
	}

	return config, nil
}

// parseAppTraceConfig 解析 application.yaml 中的 trace 配置
// application.yaml 格式：
//   trace:
//     enabled: true
//     service_name: my_service
//     exporter:
//       type: otlp
func (l *ConfigLoader) parseAppTraceConfig(v *viper.Viper) (*TraceConfig, error) {
	serviceName := detectServiceName()

	// 尝试从 app.name 读取服务名
	if v.IsSet("app.name") {
		serviceName = v.GetString("app.name")
	}

	config := l.getDefaultConfig()
	config.ServiceName = serviceName

	// 从 trace 配置项读取
	if v.IsSet("trace.enabled") {
		config.Enabled = v.GetBool("trace.enabled")
	}
	if v.IsSet("trace.service_name") {
		config.ServiceName = v.GetString("trace.service_name")
	}
	if v.IsSet("trace.sampler.type") {
		config.Sampler.Type = v.GetString("trace.sampler.type")
	}
	if v.IsSet("trace.sampler.ratio") {
		config.Sampler.Ratio = v.GetFloat64("trace.sampler.ratio")
	}
	if v.IsSet("trace.exporter.type") {
		config.Exporter.Type = v.GetString("trace.exporter.type")
	}
	if v.IsSet("trace.exporter.otlp.endpoint") {
		config.Exporter.OTLP.Endpoint = v.GetString("trace.exporter.otlp.endpoint")
	}
	if v.IsSet("trace.exporter.otlp.timeout") {
		config.Exporter.OTLP.Timeout = v.GetInt("trace.exporter.otlp.timeout")
	}
	if v.IsSet("trace.exporter.otlp.insecure") {
		config.Exporter.OTLP.Insecure = v.GetBool("trace.exporter.otlp.insecure")
	}
	if v.IsSet("trace.exporter.max_queue_size") {
		config.Exporter.MaxQueueSize = v.GetInt("trace.exporter.max_queue_size")
	}
	if v.IsSet("trace.batch.batch_size") {
		config.Batch.BatchSize = v.GetInt("trace.batch.batch_size")
	}
	if v.IsSet("trace.batch.timeout") {
		config.Batch.Timeout = v.GetInt("trace.batch.timeout")
	}
	if v.IsSet("trace.batch.max_queue_size") {
		config.Batch.MaxQueueSize = v.GetInt("trace.batch.max_queue_size")
	}

	// 验证配置
	if err := validateConfig(config); err != nil {
		return nil, err
	}

	return config, nil
}

// getDefaultConfig 获取默认配置
func (l *ConfigLoader) getDefaultConfig() *TraceConfig {
	serviceName := detectServiceName()
	return &TraceConfig{
		Enabled:     true,
		ServiceName: serviceName,
		Sampler: SamplerConfig{
			Type:  "always_on",
			Ratio: 1.0,
		},
		Exporter: ExporterConfig{
			Type:        "stdout", // 默认输出到日志（降级模式）
			MaxQueueSize: 2048,
			OTLP: OTLPConfig{
				Endpoint: "localhost:4317",
				Timeout:  10,
				Insecure: true,
			},
		},
		Batch: BatchConfig{
			BatchSize:    512,
			Timeout:      5,
			MaxQueueSize: 2048,
		},
	}
}

// ============================================================================
// 配置加载（旧实现，保留用于向后兼容）
// ============================================================================

// LoadConfigLegacy 从配置文件加载配置（旧实现）
// 支持从多个位置读取配置文件：
//   1. ./zltrace.yaml (当前目录)
//   2. $ZLTRACE_CONFIG 环境变量指定的路径
//   3. /etc/zltrace/config.yaml (系统配置目录)
//   4. 如果都没有，返回默认配置
// 已弃用：请使用 ConfigLoader.Load() 代替
func LoadConfigLegacy() (*TraceConfig, error) {
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
// 优先级: 环境变量 > 可执行文件名 > 当前目录名 > 默认值
func detectServiceName() string {
	// 方式1: 从环境变量读取
	if name := os.Getenv("SERVICE_NAME"); name != "" {
		return name
	}
	if name := os.Getenv("APP_NAME"); name != "" {
		return name
	}

	// 方式2: 从可执行文件名获取
	if path, err := os.Executable(); err == nil {
		name := filepath.Base(path)
		// 去掉.exe后缀（Windows）
		name = strings.TrimSuffix(name, ".exe")
		if name != "" && name != "go" && name != "main" {
			return name
		}
	}

	// 方式3: 从当前目录名获取
	if dir, err := os.Getwd(); err == nil {
		name := filepath.Base(dir)
		if name != "" && name != "/" && name != "." {
			return name
		}
	}

	// 方式4: 使用默认值
	return "zltrace"
}

// detectEnv 自动检测环境名称
// 优先级: ENV > APP_ENV > GO_ENV > MODE > 默认 dev
func detectEnv() string {
	if env := os.Getenv("ENV"); env != "" {
		return env
	}
	if env := os.Getenv("APP_ENV"); env != "" {
		return env
	}
	if env := os.Getenv("GO_ENV"); env != "" {
		return env
	}
	if mode := os.Getenv("MODE"); mode != "" {
		return mode
	}
	return "dev"
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
# ========== 追踪系统配置（独立配置文件：trace.yaml） ==========
# 此文件应放在项目根目录或 resource/ 目录下

# 是否启用追踪（总开关）
enabled: true

# 服务名称（用于追踪系统中标识本服务）
# 优先级：此配置 > 环境变量 SERVICE_NAME > 自动检测
service_name: my_service

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

  # 最大队列大小
  max_queue_size: 2048

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

// GetApplicationExampleConfig 返回集成到 application.yaml 的配置示例
func GetApplicationExampleConfig() string {
	return `
# ========== 追踪系统配置（集成到 application.yaml） ==========

app:
  name: my_service  # 服务名称（zltrace 会自动读取）

# 追踪配置
trace:
  # 是否启用追踪（总开关）
  enabled: true

  # 服务名称（可选，如果不配置则使用 app.name）
  service_name: ${app.name}

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

    # 最大队列大小
    max_queue_size: 2048

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
