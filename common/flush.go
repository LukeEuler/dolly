package common

import (
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/LukeEuler/dolly/log"
)

const (
	blue = 36
)

var (
	commit    = "not set"
	buildTime = "not set"

	mutex              sync.RWMutex
	heartBeatLogFields = logrus.Fields{
		"tags":       "heart_beat",
		"commit":     commit,
		"build time": buildTime,
	}
)

func ShowVersion() {
	fmt.Printf("\x1b[%dm%s\x1b[0m %s\n", blue, "commit:    ", commit)
	fmt.Printf("\x1b[%dm%s\x1b[0m %s\n", blue, "build time:", buildTime)
}

const (
	minDuration time.Duration = -1 << 63
)

// Flush 心跳日志
func Flush(shutdown chan struct{}) {
	timer := time.NewTimer(minDuration)
	interval := time.Minute
	for {
		select {
		case <-shutdown:
			log.Entry.Warnln("stop flush")
			return
		case <-timer.C:
			log.Entry.WithFields(heartBeatLogFields).Info()
			timer.Reset(interval)
		}
	}
}

/*
在 extra 添加额外信息
*/
func AddHeartBeatField(key string, value any) {
	mutex.Lock()
	defer mutex.Unlock()
	temp, ok := heartBeatLogFields["extra"]
	if !ok {
		temp = make(logrus.Fields)
	}
	temp.(logrus.Fields)[key] = value
	heartBeatLogFields["extra"] = temp
}
