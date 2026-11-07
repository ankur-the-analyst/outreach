package config

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Addr            string `default:":8080"`
	DatabaseAppURL  string `required:"true" split_words:"true"`
	MasterKeys      string `required:"true" split_words:"true"`
	MasterKeyActive int    `split_words:"true" default:"1"`
	UndoHoldS       int    `split_words:"true" default:"300"`
	UnknownTimeoutS int    `split_words:"true" default:"900"`
}

func Load() (Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return Config{}, fmt.Errorf("config: %w", err)
	}
	return cfg, nil
}
