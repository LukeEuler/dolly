package common

import (
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/LukeEuler/dolly/log"
)

func DoLoopJobs(jobs ...func(chan struct{})) {
	shutdown := make(chan struct{})
	// Why does signal.Notify use buffered channel? https://www.sobyte.net/post/2022-06/signal-channel/
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	go func() {
		for sig := range signals {
			log.Entry.Infof("received signal [%v], preparing to quit", sig)
			close(shutdown)
		}
	}()

	var wg sync.WaitGroup

	for index := range jobs {
		wg.Add(1)
		job := jobs[index]
		go func() {
			defer wg.Done()
			job(shutdown)
		}()
	}

	wg.Wait()
}
