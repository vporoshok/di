package di

import (
	"context"
	"errors"
	"fmt"
	"reflect"
)

type Container interface {
	RegisterStruct(name string, service interface{}, opts ...Option)
	// RegisterFunc(name string, constructor interface{}, opts ...Option)
	Check(context.Context) error
	Get(ctx context.Context, name string, dst interface{}) error
	// MustGet(ctx context.Context, name string, dst interface{})
	// Make(ctx context.Context, constructor interface{}) (interface{}, error)
	// MustMake(ctx context.Context, constructor interface{}) interface{}
}

func NewContainer() Container {
	return &container{
		singletones:  map[string]interface{}{},
		constructors: map[string]func(context.Context, Container) (interface{}, error){},
	}
}

type container struct {
	err          error
	locked       bool
	singletones  map[string]interface{}
	constructors map[string]func(ctx context.Context, dc Container) (interface{}, error)
}

func (dc *container) RegisterStruct(name string, service interface{}, opts ...Option) {
	if dc.locked || dc.err != nil {
		return
	}
	orig := reflect.TypeOf(service)
	t := orig
	if t.Kind() != reflect.Struct {
		dc.err = fmt.Errorf("service should be an struct or pointer to struct, but got %T", service)
		return
	}
	m := make(map[int]string, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		token := field.Tag.Get("di")
		if token != "" {
			m[i] = token
		}
	}
	dc.addConstructor(name, func(ctx context.Context, _ Container) (interface{}, error) {
		res := reflect.New(orig)
		v := res
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}
		for i, token := range m {
			field := v.Field(i)
			if field.Kind() == reflect.Ptr {
				value := reflect.New(field.Type().Elem())
				field.Set(value)
			}
			if err := dc.get(ctx, token, field); err != nil {
				return nil, err
			}
		}
		return res.Interface(), nil
	}, opts...)
}

func (dc *container) RegisterFunc(name string, constructor interface{}, opts ...Option) {
	val := reflect.ValueOf(constructor)
	t := val.Type()
	// t.In()
	_ = t
}

func (dc *container) addConstructor(
	name string, constructor func(context.Context, Container) (interface{}, error), opts ...Option,
) {
	if dc.locked || dc.err != nil {
		return
	}
	if _, exists := dc.constructors[name]; exists {
		dc.err = fmt.Errorf("service %s already registered", name)
		return
	}
	cfg := config{}
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.Singletone {
		inner := constructor
		constructor = func(ctx context.Context, _ Container) (interface{}, error) {
			if res, ok := dc.singletones[name]; ok {
				return res, nil
			}
			res, err := inner(ctx, dc)
			dc.singletones[name] = res
			return res, err
		}
	}
	dc.constructors[name] = constructor
}

func (dc *container) Check(ctx context.Context) error {
	if dc.locked || dc.err != nil {
		return dc.err
	}
	dc.locked = true
	for name, constructor := range dc.constructors {
		if _, err := constructor(ctx, dc); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	return nil
}

func (dc *container) Get(ctx context.Context, name string, dst interface{}) error {
	if !dc.locked {
		return errors.New("container should be locked")
	}
	return dc.get(ctx, name, reflect.ValueOf(dst).Elem())
}

func (dc *container) get(ctx context.Context, name string, dstValue reflect.Value) error {
	constructor, ok := dc.constructors[name]
	if !ok {
		return fmt.Errorf("service %s not found", name)
	}
	// TODO: detect cycle dependency
	service, err := constructor(ctx, dc)
	if err != nil {
		return err
	}
	if dstValue.Kind() != reflect.Interface && dstValue.IsNil() {
		return fmt.Errorf("dst must be a non nil pointer, got %s", dstValue.Kind())
	}
	srvValue := reflect.ValueOf(service)
	dstType := dstValue.Type()
	if dstType.Kind() == reflect.Interface {
		if !srvValue.Type().Implements(dstType) {
			return fmt.Errorf("service %s not implements dst interface", name)
		}
		dstValue.Set(srvValue)
	} else {
		if !srvValue.Type().AssignableTo(dstType) {
			return fmt.Errorf("service %s (%T) not assignable to dst type (%s)",
				name, service, dstType.Name())
		}
		dstValue.Set(srvValue)
	}
	return nil
}
