package zltrace

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// 测试默认配置
	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if config == nil {
		t.Fatal("config should not be nil")
	}

	if !config.Enabled {
		t.Error("enabled should be true by default")
	}

	if config.ServiceName == "" {
		t.Error("service name should not be empty")
	}

	if config.Exporter.Type == "" {
		t.Error("exporter type should not be empty")
	}
}

func TestDetectServiceName(t *testing.T) {
	// 测试环境变量
	os.Setenv("SERVICE_NAME", "test-service")
	name := detectServiceName()
	if name != "test-service" {
		t.Errorf("expected test-service, got %s", name)
	}
	os.Unsetenv("SERVICE_NAME")

	// 测试默认值
	// 注意：在测试环境中，可执行文件名可能是 zltrace.test 或 zltrace.exe
	// 所以我们只检查是否包含 "zltrace"
	name = detectServiceName()
	// 在测试环境中，可执行文件名会被检测到（如 zltrace.test）
	// 只要包含 "zltrace" 就算通过
	if name != "zltrace" && name != "zltrace.test" && name != "service" {
		t.Errorf("expected zltrace or zltrace.test, got %s", name)
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *TraceConfig
		wantErr bool
	}{
		{
			name: "valid config - otlp",
			config: &TraceConfig{
				Enabled:     true,
				ServiceName: "test",
				Exporter: ExporterConfig{
					Type: "otlp",
					OTLP: OTLPConfig{
						Endpoint: "localhost:4317",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid config - stdout",
			config: &TraceConfig{
				Enabled:     true,
				ServiceName: "test",
				Exporter: ExporterConfig{
					Type: "stdout",
				},
			},
			wantErr: false,
		},
		{
			name: "valid config - none",
			config: &TraceConfig{
				Enabled:     true,
				ServiceName: "test",
				Exporter: ExporterConfig{
					Type: "none",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid exporter type",
			config: &TraceConfig{
				Enabled:     true,
				ServiceName: "test",
				Exporter: ExporterConfig{
					Type: "invalid",
				},
			},
			wantErr: true,
		},
		{
			name: "otlp without endpoint",
			config: &TraceConfig{
				Enabled:     true,
				ServiceName: "test",
				Exporter: ExporterConfig{
					Type: "otlp",
					OTLP: OTLPConfig{
						Endpoint: "",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "disabled trace",
			config: &TraceConfig{
				Enabled:     false,
				ServiceName: "test",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
