package dragonfly

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/LukeEuler/dolly/common/tree"
)

func testNextClient() (tree.Context, error) {
	return nil, nil
}

func func1(_ tree.Context, in []int) ([]int64, error) {
	result := make([]int64, 0, len(in))
	for _, v := range in {
		result = append(result, int64(v+1000))
	}
	return result, nil
}

func TestNewDragonfly(t *testing.T) {
	d := NewDragonfly(2, 1, 3, 1, NewWorkerFactory(testNextClient, func1))

	inputs := make([]int, 0, 100)
	for i := 0; i < 20; i++ {
		inputs = append(inputs, i)
	}
	result, err := d.Get(inputs)
	assert.NoError(t, err)
	assert.Len(t, result, 20)

	result, err = d.Get([]int{1, 2, 3, 11, 12, 13, 14})
	assert.NoError(t, err)
	assert.Len(t, result, 7)
}
