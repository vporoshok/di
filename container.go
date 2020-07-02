package di

import (
	"context"
	"errors"
	"fmt"
	"reflect"
)

type Container interface {
	// Register(name string, service interface{}, opts ...Option) error
	RegisterFunc(name string, constructor func(context.Context, Container) (interface{}, error), opts ...Option) error
	Get(ctx context.Context, name string, dst interface{}) error
}

func NewContainer() Container {
	return &container{
		constructors: map[string]func(context.Context, Container) (interface{}, error){},
	}
}

type container struct {
	constructors map[string]func(ctx context.Context, dc Container) (interface{}, error)
}

// func (dc *container) Register(name string, service interface{}, opts ...Option) error {
// 	return dc.RegisterFunc(name, func(ctx context.Context, dc container) (interface{}, error) {
// 		// TODO: default constructor
// 	}, opts...)
// }

func (dc *container) RegisterFunc(
	name string, constructor func(context.Context, Container) (interface{}, error), opts ...Option,
) error {
	if _, exists := dc.constructors[name]; exists {
		return fmt.Errorf("service %s already registered", name)
	}
	dc.constructors[name] = constructor
	return nil
}

func (dc *container) Get(ctx context.Context, name string, dst interface{}) error {
	constructor, ok := dc.constructors[name]
	if !ok {
		return fmt.Errorf("service %s not found", name)
	}
	// TODO: detect cycle dependency
	// TODO: singletone services
	service, err := constructor(ctx, dc)
	if err != nil {
		return err
	}
	dstValue := reflect.ValueOf(dst)
	if dstValue.Kind() != reflect.Ptr || dstValue.IsNil() {
		return errors.New("dst must be a non nil pointer")
	}
	srvValue := reflect.ValueOf(service)
	dstType := dstValue.Type().Elem()
	if dstType.Kind() == reflect.Interface {
		if !srvValue.Type().Implements(dstType) {
			return fmt.Errorf("service %s not implements dst interface", name)
		}
	} else {
		if !srvValue.Type().AssignableTo(dstType) {
			return fmt.Errorf("service %s not assignable to dst type", name)
		}
	}
	dstValue.Elem().Set(srvValue)
	return nil
}
