package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsNil(t *testing.T) {
	assert.True(t, IsNil(nil))

	type testInf interface {
	}
	var a testInf
	assert.True(t, IsNil(a))

	type testStruct struct {
	}
	b := new(testStruct)
	assert.False(t, IsNil(b))

	var c testStruct
	assert.False(t, IsNil(c))
}
