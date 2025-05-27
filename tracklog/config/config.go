package config

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

// global config
type GlobalConfig struct {
	MaxGoroutines   int    `yaml:"max_goroutines"`
	CPULimitPercent int    `yaml:"cpu_limit_percent"`
	MemoryLimitMB   int    `yaml:"memory_limit_mb"`
	EventTCPEnabled bool   `yaml:"event_tcp_enabled"`
	EventTCPAddress string `yaml:"event_tcp_address"`
	DefaultCompress bool   `yaml:"default_compress"`
	DefaultSavePath string `yaml:"default_save_path"`
}

// monitor config
type MonitorConfig struct {
	Name            string `yaml:"name"`
	Path            string `yaml:"path"`
	FilenamePattern string `yaml:"filename_pattern"`
	ExtractType     string `yaml:"extract_type"` // regex
	ExtractPattern  string `yaml:"extract_pattern"`
	SaveExtracted   bool   `yaml:"save_extracted"`
	SavePath        string `yaml:"save_path"`
	CompressOutput  bool   `yaml:"compress_output"`
	Realtime        bool   `yaml:"realtime"`
	EventOnNoUpdate bool   `yaml:"event_on_no_update"` // 행 걸림 감지
	EventTCPEnabled bool   `yaml:"event_tcp_enabled"`
}

type Config struct {
	Global   GlobalConfig    `yaml:"global"`
	Monitors []MonitorConfig `yaml:"monitors"`
}

// load yaml file
func LoadConfig(configPath string) (*Config, error) {
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("설정 파일 읽기 실패: %w", err)
	}

	var cfg Config
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, fmt.Errorf("YAML 파싱 실패: %w", err)
	}

	return &cfg, nil
}
