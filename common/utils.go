package common

import (
	"runtime"
	"time"

	"github.com/LukeEuler/dolly/log"
)

// TimeConsume provides convenience function for time-consuming calculation
func TimeConsume(start time.Time) {
	pc, _, _, ok := runtime.Caller(1)
	if !ok {
		return
	}

	// get Fun object from pc
	funcName := runtime.FuncForPC(pc).Name()
	log.Entry.WithField("tags", "func_time_consume").
		WithField("cost", time.Since(start).String()).Debug(funcName)
}
