package tree

import (
	"github.com/pkg/errors"
)

type Context interface {
	Set(key string, value any)
	get(key string) (any, bool)
	Clear()
}

/*
当前golang泛型不支持不同类型的出现在struct上, 因此用独立函数做功能的补充

	type Context interface {
		Set(key string, value any)
		Get[T any](key string) (T, error)
		Clear()
	}
*/
func Get[T any](ctx Context, key string) (T, error) {
	var zero T
	v, ok := ctx.get(key)
	if !ok {
		return zero, errors.Errorf("no key [%s]", key)
	}
	res, ok := v.(T)
	if !ok {
		return zero, errors.Errorf("type mismatch for key [%s]", key)
	}
	return res, nil
}

type DefaultContext struct {
	content map[string]any
}

func (c *DefaultContext) Set(key string, value any) {
	c.content[key] = value
}

func (c *DefaultContext) get(key string) (any, bool) {
	value, ok := c.content[key]
	return value, ok
}

func (c *DefaultContext) Clear() {
	c.content = make(map[string]any, 10)
}

func NewDefaultContext() Context {
	return &DefaultContext{
		content: make(map[string]any, 10),
	}
}
