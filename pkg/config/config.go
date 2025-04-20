package config

import "time"

type Config struct {
	Server struct {
		Port         string        `envconfig:"PORT" default:"8080"`
		ReadTimeout  time.Duration `envconfig:"READ_TIMEOUT" default:"30s"`
		WriteTimeout time.Duration `envconfig:"WRITE_TIMEOUT" default:"30s"`
	}
	Elasticsearch struct {
		URL         string `envconfig:"ES_URL" default:"http://localhost:9200"`
		IndexPrefix string `envconfig:"ES_INDEX_PREFIX" default:"manga_"`
	}
	Browser struct {
		UserAgent string        `envconfig:"BROWSER_UA"`
		Timeout   time.Duration `envconfig:"BROWSER_TIMEOUT" default:"30s"`
		Headless  bool          `envconfig:"BROWSER_HEADLESS" default:"true"`
	}
}

func Load() (*Config, error) {
	var cfg Config
	// Load from env
	return &cfg, nil
}
