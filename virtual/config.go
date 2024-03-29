package virtual

import (
	"errors"
	"fmt"
	"io/fs"

	"github.com/pelletier/go-toml/v2"
)

// Config contains configuration data from the whisper.cfg file.
type Config struct {
	Expires       Duration          `toml:"expires"`       // Expiry duration for dynamic content
	StaticExpires Duration          `toml:"staticexpires"` // Expiry duration for static content
	CacheSize     int               `toml:"cachesize"`     // Cache size in megabytes
	CacheDuration Duration          `toml:"cacheduration"` // Cache duration
	Headers       map[string]string `toml:"headers"`       // Headers to add
}

// Config returns configuration from the whisper.cfg file.
// It is not an error if the file does not exist.
func (vfs *FS) Config() (*Config, error) {
	var cfg Config
	cfgBytes, err := fs.ReadFile(vfs.fs, "whisper.cfg")
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return &cfg, nil
		}
		return nil, fmt.Errorf("cannot read config file: %w", err)
	} else {
		err = toml.Unmarshal(cfgBytes, &cfg)
		if err != nil {
			return nil, fmt.Errorf("cannot parse config file: %w", err)
		}
	}
	return &cfg, nil
}
