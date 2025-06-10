package tentacle

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func testNewFactory(f func(int, int) (any, error)) Factory {
	// 函数变量，
	salt := 110
	tenFailed := false
	return func() (Worker, error) {
		// xxxx
		return func(inputs chan int, outputs chan *Box) {
			for {
				sequence := <-inputs
				res, err := f(salt, sequence)
				if sequence == 10 && !tenFailed {
					tenFailed = true
					err = errors.New("haha")
				}
				if sequence%3 == 0 {
					time.Sleep(time.Millisecond * 3)
				}
				outputs <- &Box{
					Sequence: sequence,
					Result:   res,
					Err:      err,
				}
			}
		}, nil
	}
}

func testHandleSequenceV1(salt int, sequence int) (any, error) {
	return salt + sequence, nil
}

func TestNewTentacle(t *testing.T) {
	tentacle := NewTentacle(3, 2, 4, testNewFactory(testHandleSequenceV1))
	err := tentacle.UpdateMaxSequence(10)
	assert.NoError(t, err)

	value, err := tentacle.Get(1)
	assert.NoError(t, err)
	assert.Equal(t, 111, value)

	_, err = tentacle.Get(6)
	assert.Error(t, err)

	value, err = tentacle.Get(1)
	assert.NoError(t, err)
	assert.Equal(t, 111, value)

	_, err = tentacle.Get(11)
	assert.Error(t, err)

	_, err = tentacle.Get(0)
	assert.Error(t, err)

	err = tentacle.UpdateMaxSequence(9)
	assert.Error(t, err)

	value, err = tentacle.Get(2)
	assert.NoError(t, err)
	assert.Equal(t, 112, value)

	value, err = tentacle.Get(3)
	assert.NoError(t, err)
	assert.Equal(t, 113, value)

	value, err = tentacle.Get(4)
	assert.NoError(t, err)
	assert.Equal(t, 114, value)

	value, err = tentacle.Get(5)
	assert.NoError(t, err)
	assert.Equal(t, 115, value)

	value, err = tentacle.Get(6)
	assert.NoError(t, err)
	assert.Equal(t, 116, value)

	value, err = tentacle.Get(7)
	assert.NoError(t, err)
	assert.Equal(t, 117, value)

	value, err = tentacle.Get(8)
	assert.NoError(t, err)
	assert.Equal(t, 118, value)

	value, err = tentacle.Get(9)
	assert.NoError(t, err)
	assert.Equal(t, 119, value)

	value, err = tentacle.Get(10)
	assert.NoError(t, err)
	assert.Equal(t, 120, value)

	err = tentacle.UpdateMaxSequence(12)
	assert.NoError(t, err)
}

func TestNewTentacleV2(t *testing.T) {
	tentacle := NewTentacle(3, 2, 1, testNewFactory(testHandleSequenceV1))
	err := tentacle.UpdateMaxSequence(5)
	assert.NoError(t, err)

	value, err := tentacle.Get(1)
	assert.NoError(t, err)
	assert.Equal(t, 111, value)

	value, err = tentacle.Get(2)
	assert.NoError(t, err)
	assert.Equal(t, 112, value)

	value, err = tentacle.Get(3)
	assert.NoError(t, err)
	assert.Equal(t, 113, value)

	value, err = tentacle.Get(4)
	assert.NoError(t, err)
	assert.Equal(t, 114, value)

	_, err = tentacle.Get(2)
	assert.Error(t, err)

	tentacle.Stop()

	err = tentacle.UpdateMaxSequence(4)
	assert.NoError(t, err)

	value, err = tentacle.Get(2)
	assert.NoError(t, err)
	assert.Equal(t, 112, value)

	value, err = tentacle.Get(3)
	assert.NoError(t, err)
	assert.Equal(t, 113, value)

	value, err = tentacle.Get(4)
	assert.NoError(t, err)
	assert.Equal(t, 114, value)
}
