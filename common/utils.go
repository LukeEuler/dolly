package common

import (
	"reflect"
	"runtime"
	"strings"
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

// IsNil checks if a specified object is nil or not, without Failing.
// from: https://github.com/stretchr/testify/blob/master/assert/assertions.go#L647
func IsNil(object any) bool {
	if object == nil {
		return true
	}

	value := reflect.ValueOf(object)
	kind := value.Kind()
	isNilableKind := containsKind(
		[]reflect.Kind{
			reflect.Chan, reflect.Func,
			reflect.Interface, reflect.Map,
			reflect.Ptr, reflect.Slice, reflect.UnsafePointer},
		kind)

	if isNilableKind && value.IsNil() {
		return true
	}

	return false
}

// containsKind checks if a specified kind in the slice of kinds.
func containsKind(kinds []reflect.Kind, kind reflect.Kind) bool {
	for i := 0; i < len(kinds); i++ {
		if kind == kinds[i] {
			return true
		}
	}

	return false
}

func FuncName(f any) string {
	funcName := runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
	list := strings.Split(funcName, "/")
	if len(list) > 0 {
		funcName = list[len(list)-1]
	}
	return funcName
}

// FormatHexString 0XaBcDef0123456789 -> abcdef0123456789
func FormatHexString(content string) string {
	return strings.TrimPrefix(strings.ToLower(content), "0x")
}
