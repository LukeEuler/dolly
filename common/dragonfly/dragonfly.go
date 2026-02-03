package dragonfly

import (
	"context"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/LukeEuler/dolly/log"
)

/*
TODO
inputs, outputs 目前是每次Get时, 创建并销毁
应该升级成长期存活的状态
*/
type Dragonfly[T any, R any] struct {
	inUse bool
	mutex sync.RWMutex

	// worker 工厂，将具体的业务逻辑从 dragonfly 中剥离
	generate Factory[T, R]

	// 不变参数在此设置
	concurrent int // worker 同时运作数
	redundancy int
	split      int
	maxTry     int
	workLength int // 通用工作空间，inputs,outputs,queue 的大小

	// 每次运行都需要初始化
	inputs  chan *box[T, R]
	outputs chan *box[T, R]
	cursor  cursor
}

type cursor struct {
	idx int
	len int
}

func NewDragonfly[T any, R any](concurrent, redundancy, split, maxTry int, wf Factory[T, R]) *Dragonfly[T, R] {
	concurrent = max(concurrent, 1)
	redundancy = max(redundancy, 0)
	split = max(split, 1)
	maxTry = max(maxTry, 1)
	workLength := concurrent + redundancy

	return &Dragonfly[T, R]{
		concurrent: concurrent,
		redundancy: redundancy,
		split:      split,
		maxTry:     maxTry,
		workLength: workLength,
		generate:   wf,
		cursor:     cursor{},
	}
}

func (d *Dragonfly[T, R]) Get(list []T) ([]R, error) {
	if len(list) == 0 {
		return []R{}, nil
	}
	start := time.Now()
	d.mutex.Lock()
	defer d.mutex.Unlock()
	if d.inUse {
		return nil, errors.New("in use")
	}

	d.inUse = true
	result, err := d.get(list)
	d.inUse = false
	log.Entry.
		WithField("dragonfly", "GET").
		WithField("cost", time.Since(start).String()).
		Infof("get %d", len(list))
	return result, err
}

func (d *Dragonfly[T, R]) get(list []T) ([]R, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	d.init(list)
	defer d.clearChannels()

	var wg sync.WaitGroup
	wg.Add(d.concurrent)

	for i := 0; i < d.concurrent; i++ {
		worker, err := d.generate()
		if err != nil {
			cancel()
			return nil, err
		}
		go func(w Worker[T, R]) {
			defer wg.Done()
			w(ctx, d.inputs, d.outputs)
		}(worker)
	}

	var finalErr error
	result := make([]R, 0, d.cursor.len)
	for item := range d.outputs {
		item.count++
		if item.Err != nil {
			log.Entry.WithError(item.Err).Error(item.Err)
			if item.count > d.maxTry {
				finalErr = item.Err
				cancel()
				break
			}
			d.inputs <- item
			continue
		}
		result = append(result, item.Result...)
		if d.cursor.idx >= d.cursor.len {
			// 不再需要添加任务
			if len(result) >= d.cursor.len {
				// 任务全部完成
				break
			}
			continue
		}
		j := d.cursor.idx + d.split
		j = min(j, d.cursor.len)
		d.inputs <- &box[T, R]{
			count: 1,
			start: d.cursor.idx,
			end:   j,
			total: d.cursor.len,
			In:    list[d.cursor.idx:j],
		}
		d.cursor.idx += d.split
	}

	if finalErr != nil {
		return nil, finalErr
	}
	return result, nil
	// return result, nil
}

func (d *Dragonfly[T, R]) init(list []T) {
	d.cursor.idx = 0
	d.cursor.len = len(list)
	d.inputs = make(chan *box[T, R], d.workLength)
	d.outputs = make(chan *box[T, R], d.workLength)

	for k := 0; k < d.workLength; k++ {
		if d.cursor.idx >= d.cursor.len {
			break
		}
		j := d.cursor.idx + d.split
		if j > d.cursor.len {
			j = d.cursor.len
		}
		d.inputs <- &box[T, R]{
			count: 1,
			start: d.cursor.idx,
			end:   j,
			total: d.cursor.len,
			In:    list[d.cursor.idx:j],
		}
		d.cursor.idx += d.split
	}
}

func (d *Dragonfly[T, R]) clearChannels() {
	// close(d.outputs)
	close(d.inputs)
}
