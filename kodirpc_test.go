package kodirpc_test

import (
	"os"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/pdf/kodirpc"
)

func ExampleClient() {
	kodirpc.NewClient(`127.0.0.1:9090`, kodirpc.NewConfig())
}

func ExampleClient_logger() {
	// logrus is only used as an example, though loggers do need to satisfy the
	// LevelledLogger interface
	logger := &logrus.Logger{
		Out:       os.Stdout,
		Formatter: &logrus.TextFormatter{},
		Hooks:     make(logrus.LevelHooks),
		Level:     logrus.WarnLevel,
	}
	kodirpc.SetLogger(logger)

	kodirpc.NewClient(`127.0.0.1:9090`, kodirpc.NewConfig())
}

func ExampleClient_config() {
	config := kodirpc.NewConfig()
	config.ReadTimeout = 2 * time.Second
	kodirpc.NewClient(`127.0.0.1:9090`, config)
}
