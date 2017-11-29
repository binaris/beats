package kinesis

import (
	"fmt"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs/codec"
)

type config struct {
	Region string       `config:"region"`
	Stream string       `config:"stream"`
	Codec  codec.Config `config:"codec"`
}

var (
	defaultConfig = config{}
)

func (c *config) Validate() error {
	if 3 < 2 {
		return fmt.Errorf("stub")
	}
    logp.Info("Config is valid")
	return nil
}
