package kodirpc_test

import (
	"fmt"
	"os"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/pdf/kodirpc"
)

func ExampleClient() {
	client, err := kodirpc.NewClient(`127.0.0.1:9090`, kodirpc.NewConfig())
	if err != nil || client == nil {
		panic(err)
	}
}

func ExampleClient_logger() {
	// logrus is only used as an example, though loggers do need to satisfy the
	// LevelledLogger interface.
	logger := &logrus.Logger{
		Out:       os.Stdout,
		Formatter: &logrus.TextFormatter{},
		Hooks:     make(logrus.LevelHooks),
		Level:     logrus.WarnLevel,
	}
	kodirpc.SetLogger(logger)

	client, err := kodirpc.NewClient(`127.0.0.1:9090`, kodirpc.NewConfig())
	if err != nil || client == nil {
		panic(err)
	}
}

func ExampleClient_config() {
	config := kodirpc.NewConfig()
	config.ReadTimeout = 2 * time.Second

	client, err := kodirpc.NewClient(`127.0.0.1:9090`, config)
	if err != nil || client == nil {
		panic(err)
	}
}

func ExampleClient_Call() {
	client, err := kodirpc.NewClient(`127.0.0.1:9090`, kodirpc.NewConfig())
	if err != nil {
		panic(err)
	}
	defer func() {
		if err = client.Close(); err != nil {
			panic(err)
		}
	}()

	params := map[string]interface{}{
		`title`:       `Hello`,
		`message`:     `From kodirpc`,
		`displaytime`: 5000,
	}
	res, err := client.Call(`GUI.ShowNotification`, params)
	if err != nil {
		panic(err)
	}

	fmt.Println(res)
}

func ExampleClient_Notify() {
	client, err := kodirpc.NewClient(`127.0.0.1:9090`, kodirpc.NewConfig())
	if err != nil {
		panic(err)
	}
	defer func() {
		if err = client.Close(); err != nil {
			panic(err)
		}
	}()

	params := map[string]interface{}{
		`title`:       `Hello`,
		`message`:     `From kodirpc`,
		`displaytime`: 5000,
	}
	err = client.Notify(`GUI.ShowNotification`, params)
	if err != nil {
		panic(err)
	}
}

func ExampleClient_Handle() {
	client, err := kodirpc.NewClient(`127.0.0.1:9090`, kodirpc.NewConfig())
	if err != nil {
		panic(err)
	}
	defer func() {
		if err = client.Close(); err != nil {
			panic(err)
		}
	}()

	// Handle will execute the handler func every time a notification with the
	// specified method is sent.
	client.Handle(`VideoLibrary.OnUpdate`, func(method string, data interface{}) {
		fmt.Println(data.(map[string]interface{}))
	})

	// Your application loop/logic should go here, the following will block
	// indefinitely
	done := make(chan struct{})
	<-done
}
