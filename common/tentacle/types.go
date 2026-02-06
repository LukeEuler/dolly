package tentacle

import (
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/LukeEuler/dolly/common/tree"
	"github.com/LukeEuler/dolly/log"
)

// Factory 是 Worker 的构造器接口方法，同时也是为了方便隐藏业务的初始化变量
type Factory[T any] func() (Worker[T], error)

// Worker 用于处理具体业务对接口方法
type Worker[T any] func(inputs chan int64, outputs chan *box[T])

// box 是 Worker 处理后的数据结果
type box[T any] struct {
	sequence int64
	result   T
	err      error
}

/*
ITentacle 是一个针对增量序列(非严格增量)顺序处理，的多协程加速工具
通常情况下，我们对事务 1，2，3。。。按顺序处理。然而，当这个序列变得很大时，处理耗时就很明显了。
存在这么一种情况：
- 每个事务都可以拆分为两个部分，A->B。
- 我们假设对一个事务，只有处理完 A 之后，才能够相应对处理 B
- 再假设 任意两个事务之间对 A 的都是相互无关的
那么，此时可以预先用协程对事务并行的处理 A 的部分，然后顺序的处理 B 部分。
假设 处理 A，B 耗时分别是 Ta, Tb。
那么，顺序处理的总耗时 T = Ta1+Tb1+Ta2+Tb2...。
优化后，耗时最少可以减少到 T = Ta1+Tb1+Tb2...

我们假定：

	当外部调用处理了某序列 s 后，不会出现小于 s 的处理，（但可能会多次处理 s）
	更严格来说：如果某次处理的序列为 s 时，那么下次的处理序列，要么是 s，要么是 s+1
*/
type ITentacle[T any] interface {
	UpdateMaxSequence(sequence int64) error // 规定当前的最大处理序列
	Get(sequence int64) (T, error)          // 按序列获取数据
	Stop()                                  // 将 Tentacle 恢复到初始状态
}

func NewWorkerFactory[T any](
	nextClient func() (tree.Context, error),
	f func(tree.Context, int64) (T, error)) Factory[T] {
	funcName := runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
	list := strings.Split(funcName, "/")
	if len(list) > 0 {
		funcName = list[len(list)-1]
	}

	return func() (Worker[T], error) {
		ctx, err := nextClient()
		if err != nil {
			return nil, err
		}
		return func(inputs chan int64, outputs chan *box[T]) {
			for {
				height, ok := <-inputs
				if !ok {
					// inputs 被关闭了, worker 就需要停下
					break
				}
				log.Entry.WithField("tentacle", funcName).Infof("try get %d", height)
				start := time.Now()
				res, err := f(ctx, height)
				log.Entry.WithField("tentacle", funcName).
					WithField("cost", time.Since(start).String()).
					Infof("%d done", height)
				if err != nil {
					// 防止程序在错误上，过多的浪费资源。主要是错误日志会爆
					time.Sleep(time.Second)
				}
				outputs <- &box[T]{
					sequence: height,
					result:   res,
					err:      err,
				}
			}
		}, nil
	}
}
