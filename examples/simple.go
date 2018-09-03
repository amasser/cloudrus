package main

import (
	"math/rand"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/zwerch/cloudrus"
)

func main() {
	log := logrus.New()
	groupName := os.Getenv("AWS_CLOUDWATCHLOGS_GROUP")
	streamName := os.Getenv("AWS_CLOUDWATCHLOGS_STREAM")

	cloudwatchHook, err := cloudrus.NewHook(
		groupName, streamName, time.Millisecond,
	)
	if err != nil {
		panic(err)
	}

	log.AddHook(cloudwatchHook)

	log.Info("starting task")

	// simulating some concurrent tasks
	for i := 1; i < 50; i++ {
		go log.WithField("index", i).Debug("this is a debug log message")
		go log.WithField("index", i).Info("this is an Info log message")
		go log.WithField("index", i).Error("this is an Error log message")
		go log.WithField("index", i).Warning("this is a Warning log message")
		time.Sleep(time.Duration(rand.Int63n(int64(i*10))) * time.Millisecond)
	}

	time.Sleep(5 * time.Second)
	log.WithField("done", true).Info("finally")
}
