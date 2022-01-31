package virtual

import (
	"errors"
	"fmt"
	"io/fs"

	"github.com/pelletier/go-toml/v2"
)

// Config contains configuration data from the whisper.cfg file.
type Config struct {
	Expires       Duration          `toml:"expires"`
	StaticExpires Duration          `toml:"staticexpires"`
	Headers       map[string]string `toml:"headers"`
}

// Config returns configuration from the whisper.cfg file.
// It is not an error if the file does not exist.
func (vfs *FS) Config() (*Config, error) {
	var cfg Config
	cfgBytes, err := fs.ReadFile(vfs.fs, "whisper.cfg")
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("Cannot read config file: %w", err)
	} else {
		err = toml.Unmarshal(cfgBytes, &cfg)
		if err != nil {
			return nil, fmt.Errorf("Cannot parse config file: %w", err)
		}
	}
	return &cfg, nil
}
