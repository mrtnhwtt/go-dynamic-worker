package config

type Config struct {
	WorkerCount int32
}

func LoadFromEnv() (*Config, error) {
	return &Config{
		WorkerCount: 20,
	}, nil
}
