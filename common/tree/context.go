package tree

import (
	"github.com/pkg/errors"
)

/*
原接口和独立泛型函数保留作迁移参考。
Go 1.27 支持具体类型的泛型方法，接口方法仍不能声明类型参数。

type Context interface {
	Set(key string, value any)
	get(key string) (any, bool)
	Clear()
}

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
*/

type Context struct {
	content map[string]any
}

func (c *Context) Set(key string, value any) {
	c.content[key] = value
}

func (c *Context) Get[T any](key string) (T, error) {
	var zero T
	value, ok := c.content[key]
	if !ok {
		return zero, errors.Errorf("no key [%s]", key)
	}
	res, ok := value.(T)
	if !ok {
		return zero, errors.Errorf("type mismatch for key [%s]", key)
	}
	return res, nil
}

func (c *Context) Clear() {
	c.content = make(map[string]any, 10)
}

func NewContext() *Context {
	return &Context{
		content: make(map[string]any, 10),
	}
}
