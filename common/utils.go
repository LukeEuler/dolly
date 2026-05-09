package common

import (
	"reflect"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/LukeEuler/dolly/log"
)

func TraceTime() func() {
	start := time.Now()
	pc, _, _, ok := runtime.Caller(1)
	if !ok {
		return func() {}
	}
	funcName := runtime.FuncForPC(pc).Name()
	return func() {
		log.Entry.WithField("tags", "func_time_consume").
			WithField("cost", time.Since(start).String()).
			Debug(funcName)
	}
}

// IsNil checks if a specified object is nil or not, without Failing.
// from: https://github.com/stretchr/testify/blob/master/assert/assertions.go#L719
func IsNil(object any) bool {
	if object == nil {
		return true
	}

	value := reflect.ValueOf(object)
	switch value.Kind() {
	case
		reflect.Chan, reflect.Func,
		reflect.Interface, reflect.Map,
		reflect.Pointer, reflect.Slice, reflect.UnsafePointer:

		return value.IsNil()
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

// CheckList 检查是否包含元素
type CheckList []string

func (l *CheckList) Contains(value string) bool {
	return slices.Contains(*l, value)
}

func (l *CheckList) AddNoChange(al *CheckList) *CheckList {
	ret := new(CheckList)
	*ret = append(*l, *al...)
	return ret
}

func (l *CheckList) DeleteNoChange(elem string) *CheckList {
	tmp := new(CheckList)
	for _, v := range *l {
		if v != elem {
			*tmp = append(*tmp, v)
		}
	}
	return tmp
}
