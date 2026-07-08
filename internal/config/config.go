package config

import (
	"log"
	"sync"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	// Path to the gallery directory where images are stored
	GalleryBasePath   string `env:"GALLERY_BASE_PATH" envDefault:"/mnt"`
	GRPCServerAddress string `env:"GRPC_SERVER_ADDRESS" envDefault:"localhost:9001"`
}

var (
	appConfig *Config
	once      sync.Once
)

func Get() *Config {
	once.Do(func() {
		appConfig = &Config{}

		if err := cleanenv.ReadConfig(".env", appConfig); err != nil {
			log.Fatalf("FATAL: Cannot read config: %v", err)
		}
	})
	return appConfig
}
