package config

// Scaffold builds a starter project Config: just the current schema version.
// Services are scaffolded into profiles (see package profile), not here.
func Scaffold() *Config {
	return &Config{Version: CurrentVersion}
}
