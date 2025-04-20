package config

import (
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Server struct {
		Port         string        `envconfig:"SERVER_PORT" default:"8080"`
		ReadTimeout  time.Duration `envconfig:"SERVER_READ_TIMEOUT" default:"30s"`
		WriteTimeout time.Duration `envconfig:"SERVER_WRITE_TIMEOUT" default:"30s"`
	}

	Elasticsearch struct {
		URL string `envconfig:"ELASTICSEARCH_URL" default:"http://elasticsearch:9200"`
	}

	Downloader struct {
		OutputFolder string `envconfig:"OUTPUT_FOLDER" default:"output"`
		UserAgent    string `envconfig:"USER_AGENT" default:"Mozilla/5.0..."`
	}
}

type BrowserConfig struct {
	UserAgent   string        `envconfig:"BROWSER_USER_AGENT"`
	Headless    bool          `envconfig:"BROWSER_HEADLESS" default:"true"`
	Timeout     time.Duration `envconfig:"BROWSER_TIMEOUT" default:"30s"`
	UserDataDir string        `envconfig:"BROWSER_USER_DATA_DIR"`
}

type DownloadConfig struct {
	OutputFolder string        `envconfig:"DOWNLOAD_OUTPUT"`
	RetryCount   int           `envconfig:"DOWNLOAD_RETRIES" default:"3"`
	DelayBetween time.Duration `envconfig:"DOWNLOAD_DELAY" default:"500ms"`
}

func Load() (*Config, error) {
	var cfg Config
	err := envconfig.Process("", &cfg)
	if err != nil {
		return nil, fmt.Errorf("config load error: %w", err)
	}
	return &cfg, nil
}
