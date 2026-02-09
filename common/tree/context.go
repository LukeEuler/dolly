package tree

import (
	"reflect"

	"github.com/pkg/errors"
)

type Context interface {
	Set(key string, value any)
	Get(key string, res any) error
	Clear()
}

func Get[T any](ctx Context, key string) (*T, error) {
	var res T
	err := ctx.Get(key, &res)
	return &res, err
}

type DefaultContext struct {
	content map[string]any
}

func (c *DefaultContext) Set(key string, value any) {
	c.content[key] = value
}

func (c *DefaultContext) Get(key string, res any) error {
	value, ok := c.content[key]
	if !ok {
		return errors.Errorf("no key [%s]", key)
	}

	dest := reflect.ValueOf(res)
	if dest.Kind() == reflect.Ptr || dest.Kind() == reflect.Interface {
		dest = dest.Elem()
	} else {
		return errors.Errorf("invalid type for key(%s): %s",
			key, dest.Kind().String())
	}

	source := reflect.ValueOf(value)

	if source.Kind() != dest.Kind() {
		return errors.Errorf("invalid type for key(%s), need %s, get %s",
			key, source.Kind().String(), dest.Kind().String())
	}
	dest.Set(source)

	return nil
}

func (c *DefaultContext) Clear() {
	c.content = make(map[string]any, 10)
}

func NewDefaultContext() Context {
	return &DefaultContext{
		content: make(map[string]any, 10),
	}
}
