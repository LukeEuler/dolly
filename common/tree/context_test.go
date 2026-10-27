package tree_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/LukeEuler/dolly/common/tree"
)

func TestGet(t *testing.T) {
	ctx := tree.NewContext()

	ctx.Set("abc", 123)
	v, err := ctx.Get[int]("abc")
	assert.NoError(t, err)
	assert.Equal(t, 123, v)

	a := &s1{
		V1: 89312,
		V2: "njsakdnalskxcas",
		V3: []bool{true, false, false, true, false},
	}
	ctx.Set("cba", a)
	b, err := ctx.Get[*s1]("cba")
	assert.NoError(t, err)
	assert.True(t, a.Equal(b))

	c := []*s1{a}
	ctx.Set("asd", c)
	d, err := ctx.Get[[]*s1]("asd")
	assert.NoError(t, err)
	e := d
	assert.Len(t, e, len(c))
	for i := range e {
		assert.True(t, e[i].Equal(c[i]))
	}

	f := &s2{
		a: 2,
		b: 3,
	}
	ctx.Clear()
	ctx.Set("abc", f)
	g, err := ctx.Get[i2]("abc")
	assert.NoError(t, err)
	assert.Equal(t, 5, g.Add())

	// ------------ error ------------
	value, err := ctx.Get[int]("abc")
	assert.EqualError(t, err, "type mismatch for key [abc]")
	assert.Zero(t, value)

	ctx.Clear()
	_, err = ctx.Get[i2]("abc")
	assert.EqualError(t, err, "no key [abc]")
	ctx.Set("reused", 42)
	reused, err := ctx.Get[int]("reused")
	assert.NoError(t, err)
	assert.Equal(t, 42, reused)

	_, err = ctx.Get[int]("123")
	assert.Error(t, err)

	_, err = ctx.Get[int]("asd")
	assert.Error(t, err)
}

type s1 struct {
	V1 int64
	V2 string
	V3 []bool
}

func (s *s1) Equal(o *s1) bool {
	if o == nil {
		return false
	}
	if s.V1 != o.V1 || s.V2 != o.V2 {
		return false
	}
	if len(s.V3) != len(o.V3) {
		return false
	}
	for i := 0; i < len(s.V3); i++ {
		if s.V3[i] != o.V3[i] {
			return false
		}
	}
	return true
}

type i2 interface {
	Add() int
}

type s2 struct {
	a, b int
}

func (s *s2) Add() int {
	return s.a + s.b
}
