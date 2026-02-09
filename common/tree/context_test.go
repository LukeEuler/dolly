package tree_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/LukeEuler/dolly/common/tree"
)

func TestGet(t *testing.T) {
	ctx := tree.NewDefaultContext()

	ctx.Set("abc", 123)
	v, err := tree.Get[int](ctx, "abc")
	assert.NoError(t, err)
	assert.Equal(t, 123, *v)

	a := &s1{
		V1: 89312,
		V2: "njsakdnalskxcas",
		V3: []bool{true, false, false, true, false},
	}
	ctx.Set("cba", *a)
	b, err := tree.Get[s1](ctx, "cba")
	assert.NoError(t, err)
	assert.True(t, a.Equal(b))

	c := []*s1{a}
	ctx.Set("asd", c)
	d, err := tree.Get[[]*s1](ctx, "asd")
	assert.NoError(t, err)
	e := *d
	assert.Len(t, e, len(c))
	for i := range e {
		assert.True(t, e[i].Equal(c[i]))
	}

	// ------------ error ------------
	_, err = tree.Get[int](ctx, "123")
	assert.Error(t, err)

	_, err = tree.Get[int](ctx, "asd")
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
