package dragonfly

import (
	"context"
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/LukeEuler/dolly/common/tree"
	"github.com/LukeEuler/dolly/log"
)

type Factory[T any, R any] func() (Worker[T, R], error)

type Worker[T any, R any] func(ctx context.Context, in chan *box[T, R], out chan *box[T, R])

type box[T any, R any] struct {
	count  int
	start  int
	end    int
	total  int
	In     []T
	Result []R
	Err    error
}

/*
竹蜻蜓

针对批量任务, 提供一个批处理接口
*/
type IDragonfly[T any, R any] interface {
	Get([]T) ([]R, error)
}

func NewWorkerFactory[T any, R any](
	nextClient func() (tree.Context, error),
	f func(tree.Context, []T) ([]R, error)) Factory[T, R] {
	funcName := runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
	list := strings.Split(funcName, "/")
	if len(list) > 0 {
		funcName = list[len(list)-1]
	}

	return func() (Worker[T, R], error) {
		tctx, err := nextClient()
		if err != nil {
			return nil, err
		}
		return func(ctx context.Context, inputs chan *box[T, R], outputs chan *box[T, R]) {
			for {
				select {
				case <-ctx.Done():
					return
				case item, ok := <-inputs:
					if !ok {
						// inputs 被关闭了, worker 就需要停下
						return
					}
					start := time.Now()
					item.Result, item.Err = f(tctx, item.In)
					log.Entry.
						WithField("dragonfly", funcName).
						WithField("cost", time.Since(start).String()).
						Infof("%d~%d %d", item.start, item.end, item.total)
					if item.Err != nil {
						item.Err = errors.Wrapf(item.Err, "%d~%d %d", item.start, item.end, item.total)
						// 防止程序在错误上，过多的浪费资源。主要是错误日志会爆
						time.Sleep(time.Second)
					}

					select {
					case outputs <- item:
					case <-ctx.Done():
						return
					}
				}
			}
		}, nil
	}
}
