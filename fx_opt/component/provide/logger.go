package provide

import (
	"encoding/json"
	"fmt"

	"github.com/luoruofeng/Naval/conf"
	"go.uber.org/zap"
)

func NewLogger(c *conf.Config) *zap.Logger {
	// For some users, the presets offered by the NewProduction, NewDevelopment,
	// and NewExample constructors won't be appropriate. For most of those
	// users, the bundled Config struct offers the right balance of flexibility
	// and convenience. (For more complex needs, see the AdvancedConfiguration
	// example.)
	//
	// See the documentation for Config and zapcore.EncoderConfig for all the
	// available options.

	format := `{
		"level": "%s",
		"encoding": "json",
		"outputPaths": ["stdout", "%s"],
		"errorOutputPaths": ["stderr"],
		"encoderConfig": {
		  "messageKey": "message",
		  "levelKey": "level",
		  "levelEncoder": "lowercase"
		}
	  }`
	rawJSON := []byte(fmt.Sprintf(format, c.LogLevel, c.LogFile))

	var cfg zap.Config
	if err := json.Unmarshal(rawJSON, &cfg); err != nil {
		panic(err)
	}
	logger := zap.Must(cfg.Build())
	defer logger.Sync()

	logger.Info("logger construction succeeded")
	return logger
}
