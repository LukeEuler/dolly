package tentacle

// my English is pool, so i doc it with Chinese

import (
	"sync"

	"github.com/pkg/errors"

	"github.com/LukeEuler/dolly/log"
)

// Tentacle implement ITentacle
type Tentacle struct {
	mutex sync.RWMutex

	// worker 工厂，将具体的业务逻辑从 tentacle 中剥离
	generate Factory

	// 不变参数在此设置
	concurrent     int64 // worker 同时运作数
	workLength     int64 // 通用工作空间，inputs,outputs,queue 的大小
	reservedLength int64 // 数据保留区长度

	/*
		变化参数在此设置，变更时要使用合适的 mutex 锁
	*/
	cursor cursor

	/*
		inputs 负责给 worker 分发任务
		outputs 负责接收 worker 的结果
		queue 负责接收 outputs 中已经完成排序的结果
	*/
	inputs  chan int64
	outputs chan *Box
	queue   chan *Box

	// outputs -> cache -> queue， 释放 outputs 空间，且在 queue 无法接收的情况下做一个缓存区
	cacheArea    map[int64]*Box
	reservedArea map[int64]any // 数据保留区，可重复查询
}

type cursor struct {
	workStarted        bool
	maxSequence        int64
	lastInputsSequence int64 // 最后一个 input sequence

	lastQueueSequence int64 // workStarted 时，记录为 sequence - 1
	reservedAreaEmpty bool
	reservedAreaMin   int64
	reservedAreaMax   int64
}

func (c *cursor) copy() cursor {
	return cursor{
		workStarted:        c.workStarted,
		maxSequence:        c.maxSequence,
		lastInputsSequence: c.lastInputsSequence,
		lastQueueSequence:  c.lastQueueSequence,
		reservedAreaEmpty:  c.reservedAreaEmpty,
		reservedAreaMin:    c.reservedAreaMin,
		reservedAreaMax:    c.reservedAreaMax,
	}
}

// NewTentacle new Tentacle
func NewTentacle(concurrent, redundancy, reservedLength int64, wf Factory) *Tentacle {
	// concurrent>=1, redundancy>=2,reservedLength>=1
	concurrent = max(concurrent, 1)
	redundancy = max(redundancy, 2)
	reservedLength = max(reservedLength, 1)
	workLength := concurrent * redundancy
	/*
		cacheArea 的 length 选用 concurrent，暂不确定是否最优。直觉设置
		reservedArea 的 length 选用 reservedLength+1 最佳
	*/
	return &Tentacle{
		concurrent:     concurrent,
		generate:       wf,
		workLength:     workLength,
		reservedLength: reservedLength,
		inputs:         make(chan int64, workLength),
		outputs:        make(chan *Box, workLength),
		queue:          make(chan *Box, workLength),

		cacheArea:    make(map[int64]*Box, concurrent),
		reservedArea: make(map[int64]any, reservedLength+1),

		cursor: cursor{
			workStarted:       false,
			reservedAreaEmpty: true,
		},
	}
}

func (t *Tentacle) Stop() {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	t.inputs = make(chan int64, t.workLength)
	t.outputs = make(chan *Box, t.workLength)
	t.queue = make(chan *Box, t.workLength)
	t.cacheArea = make(map[int64]*Box, t.concurrent)
	t.reservedArea = make(map[int64]any, t.reservedLength+1)
	t.cursor = cursor{
		workStarted:       false,
		reservedAreaEmpty: true,
	}
}

// UpdateMaxSequence implement ITentacle
func (t *Tentacle) UpdateMaxSequence(sequence int64) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	if sequence < t.cursor.maxSequence {
		return errors.Errorf("can not set max sequence as %d, while last max sequence is %d",
			sequence, t.cursor.maxSequence)
	}
	t.cursor.maxSequence = sequence
	if !t.cursor.workStarted {
		return nil
	}
	maxValue := min(t.cursor.maxSequence, t.cursor.reservedAreaMax+t.workLength)
	// 运行过程中，如果更新了 max sequence, 则可能需要加入新的任务
	for value := t.cursor.lastInputsSequence + 1; value <= maxValue; value++ {
		t.inputs <- value
		t.cursor.lastInputsSequence = value
	}
	return nil
}

func (t *Tentacle) startWork(sequence int64) error {
	t.mutex.RLock()
	startStatus := t.cursor.workStarted
	t.mutex.RUnlock()
	if startStatus {
		return nil
	}
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.cursor.reservedAreaEmpty = true
	t.cursor.reservedAreaMin = sequence - 1
	t.cursor.reservedAreaMax = sequence - 1
	t.cursor.lastQueueSequence = sequence - 1

	for i := int64(0); i < t.workLength; i++ {
		value := i + sequence
		if value <= t.cursor.maxSequence {
			t.inputs <- value
			t.cursor.lastInputsSequence = value
		}
	}

	for i := int64(0); i < t.concurrent; i++ {
		worker, err := t.generate()
		if err != nil {
			return err
		}
		go worker(t.inputs, t.outputs)
	}

	t.writeResults()
	t.cursor.workStarted = true
	return nil
}

func (t *Tentacle) copyCursor() cursor {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	return t.cursor.copy()
}

func (t *Tentacle) Get(sequence int64) (any, error) {
	// 涉及读锁
	cursorState := t.copyCursor()

	if sequence > cursorState.maxSequence {
		return nil, errors.Errorf("get sequence(%d) > max sequence(%d)", sequence, cursorState.maxSequence)
	}
	if !cursorState.reservedAreaEmpty && sequence < cursorState.reservedAreaMin {
		return nil, errors.Errorf("get sequence(%d) < min reserved(%d)", sequence, cursorState.reservedAreaMin)
	}
	if !cursorState.reservedAreaEmpty && sequence > (cursorState.reservedAreaMin+t.reservedLength) {
		return nil, errors.Errorf("get sequence(%d) > min reserved + length(%d+%d)",
			sequence, cursorState.reservedAreaMin, t.reservedLength)
	}

	// 涉及写锁
	err := t.startWork(sequence)
	if err != nil {
		return nil, err
	}

	// 涉及写锁
	return t.readFromReserved(sequence)
}

func (t *Tentacle) readFromReserved(sequence int64) (any, error) {
	for {
		// 涉及读锁
		cursorState := t.copyCursor()
		// cursorState.reservedAreaMin <= sequence <= cursorState.reservedAreaMax
		if cursorState.reservedAreaMin <= sequence && sequence <= cursorState.reservedAreaMax {
			res, ok := t.reservedArea[sequence]
			if !ok {
				return nil, errors.Errorf("something wrong: miss %d(reserved min %d, max %d)",
					sequence, cursorState.reservedAreaMin, cursorState.reservedAreaMax)
			}
			return res, nil
		}
		// 涉及写锁
		t.readOneFromQueue()
	}
}

func (t *Tentacle) readOneFromQueue() {
	newBox := <-t.queue
	nextSequence := newBox.Sequence + t.workLength
	t.mutex.Lock()
	defer t.mutex.Unlock()
	if nextSequence <= t.cursor.maxSequence && nextSequence > t.cursor.lastInputsSequence {
		/*
			注意，注释中的写法是不对的，且难以发现这个bug
			t.inputs <- nextSequence
			t.cursor.lastInputsSequence = nextSequence
		*/
		maxValue := nextSequence
		for value := t.cursor.lastInputsSequence + 1; value <= maxValue; value++ {
			t.inputs <- value
			t.cursor.lastInputsSequence = value
		}
	}

	t.cursor.reservedAreaMax = newBox.Sequence
	t.reservedArea[newBox.Sequence] = newBox.Result

	if t.cursor.reservedAreaEmpty {
		// assert newBoc.err == nil
		// t.reservedLength >= 1
		t.cursor.reservedAreaEmpty = false
		t.cursor.reservedAreaMin = newBox.Sequence
		return
	}
	// !t.cursor.reservedAreaEmpty
	// asset newBoc.sequence = t.cursor.reservedAreaMax + 1
	if (newBox.Sequence - t.cursor.reservedAreaMin) == t.reservedLength {
		delete(t.reservedArea, t.cursor.reservedAreaMin)
		t.cursor.reservedAreaMin++
	}
}

func (t *Tentacle) writeResults() {
	go func() {
		for item := range t.outputs {
			t.writeResult(item)
		}
	}()
}

func (t *Tentacle) writeResult(item *Box) {
	// 此时能够拿到数据，一定是 cursor.workStarted == true
	// 且 cursor.lastQueueSequence = sequence - 1，初始化完成了
	t.mutex.Lock()
	defer t.mutex.Unlock()
	if item.Err != nil {
		log.Entry.WithError(item.Err).Error(item.Err)
		// item.sequence <= t.cursor.maxSequence
		t.inputs <- item.Sequence
		return
	}

	// item.sequence > cursor.lastQueueSequence
	if item.Sequence > t.cursor.lastQueueSequence+1 {
		// 尚未到放入 queue 的时机，先存下来
		t.cacheArea[item.Sequence] = item
		return
	}
	// item.sequence == cursor.lastQueueSequence + 1
	index := item.Sequence + 1
	t.queue <- item
	t.cursor.lastQueueSequence = item.Sequence
	for {
		// 尝试清理缓存
		item, ok := t.cacheArea[index]
		if !ok {
			break
		}
		t.queue <- item
		delete(t.cacheArea, index)
		index++
		t.cursor.lastQueueSequence = item.Sequence
	}
}
