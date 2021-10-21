package di

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
)

type Container interface {
	RegisterStruct(name string, service interface{}, opts ...Option)
	RegisterInstance(name string, service interface{})
	RegisterFunc(name string, constructor interface{}, opts ...Option)
	Lock() error
	Check(context.Context) error
	Get(ctx context.Context, name string) (interface{}, error)
	MustGet(ctx context.Context, name string) interface{}
	Provide(ctx context.Context, name string, dst interface{}) error
	MustProvide(ctx context.Context, name string, dst interface{})
	ProvideStruct(context.Context, interface{}) error
	MustProvideStruct(context.Context, interface{})
	MustProvideHTTPHandler(ctx context.Context, constructor interface{}) http.HandlerFunc
}

func NewContainer() Container {
	dc := &container{
		singletones:  map[string]reflect.Value{},
		constructors: map[string]func(context.Context, Container) (interface{}, error){},
	}
	dc.singletones["di"] = reflect.ValueOf(dc)
	return dc
}

type container struct {
	err          error
	locked       bool
	singletones  map[string]reflect.Value
	constructors map[string]func(ctx context.Context, dc Container) (interface{}, error)
}

func (dc *container) RegisterStruct(name string, service interface{}, opts ...Option) {
	if dc.locked || dc.err != nil {
		return
	}
	orig := reflect.TypeOf(service)
	isPointer := false
	if orig.Kind() == reflect.Ptr {
		isPointer = true
		orig = orig.Elem()
	}
	if orig.Kind() != reflect.Struct {
		dc.err = fmt.Errorf("service should be an struct or pointer to struct, but got %T", service)
		return
	}
	dc.addConstructor(name, func(ctx context.Context, _ Container) (interface{}, error) {
		res := reflect.New(orig).Elem()
		v := res
		if err := dc.provideStruct(ctx, v); err != nil {
			return nil, err
		}
		if isPointer {
			res = res.Addr()
		}
		return res.Interface(), nil
	}, opts...)
}

func (dc *container) ProvideStruct(ctx context.Context, s interface{}) error {
	v := reflect.ValueOf(s)
	t := v.Type()
	if t.Kind() != reflect.Ptr || t.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("s should be an pointer to struct, but got %T", s)
	}
	return dc.provideStruct(ctx, v)
}

func (dc *container) MustProvideStruct(ctx context.Context, s interface{}) {
	if err := dc.ProvideStruct(ctx, s); err != nil {
		panic(err)
	}
}

func (dc *container) RegisterInstance(name string, service interface{}) {
	if dc.locked || dc.err != nil {
		return
	}
	if dc.isExists(name) {
		dc.err = fmt.Errorf("service %s already registered", name)
		return
	}
	dc.singletones[name] = reflect.ValueOf(service)
}

func (dc *container) RegisterFunc(name string, constructor interface{}, opts ...Option) {
	val := reflect.ValueOf(constructor)
	t := val.Type()
	if t.Kind() != reflect.Func {
		dc.err = fmt.Errorf("constructor should be an function, but got %T", constructor)
		return
	}
	if t.NumOut() != 2 || t.Out(1).Name() != "error" {
		dc.err = fmt.Errorf("constructor should return instance of service and error, but got %T", constructor)
		return
	}
	dc.addConstructor(name, func(ctx context.Context, _ Container) (interface{}, error) {
		args := make([]reflect.Value, t.NumIn())
		for i := 0; i < t.NumIn(); i++ {
			arg := t.In(i)
			if arg.PkgPath() == "context" && arg.Name() == "Context" {
				args[i] = reflect.ValueOf(ctx)
			} else {
				res := reflect.New(arg).Elem()
				v := res
				if v.Kind() == reflect.Ptr {
					v = v.Elem()
				}
				if err := dc.provideStruct(ctx, v); err != nil {
					return nil, err
				}
				args[i] = res
			}
		}
		res := val.Call(args)
		err := res[1].Interface()
		if err == nil {
			return res[0].Interface(), nil
		}
		return nil, err.(error)
	}, opts...)
}

func (dc *container) provideStruct(ctx context.Context, v reflect.Value) error {
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		tfield := t.Field(i)
		name := tfield.Tag.Get("di")
		if name == "" {
			continue
		}
		field := v.Field(i)
		if field.Kind() == reflect.Ptr {
			value := reflect.New(field.Type().Elem())
			field.Set(value)
		}
		if err := dc.provide(ctx, name, field); err != nil {
			return err
		}
	}
	return nil
}

func (dc *container) addConstructor(
	name string, constructor func(context.Context, Container) (interface{}, error), opts ...Option,
) {
	if dc.locked || dc.err != nil {
		return
	}
	if dc.isExists(name) {
		dc.err = fmt.Errorf("service %s already registered", name)
		return
	}
	cfg := config{}
	for _, opt := range opts {
		opt(&cfg)
	}
	inner := constructor
	constructor = func(ctx context.Context, _ Container) (interface{}, error) {
		if res, ok := dc.singletones[name]; ok {
			return res, nil
		}
		res, err := inner(ctx, dc)
		dc.singletones[name] = reflect.ValueOf(res)
		return res, err
	}
	dc.constructors[name] = constructor
}

func (dc *container) isExists(name string) bool {
	if _, exists := dc.constructors[name]; exists {
		return true
	}
	if _, exists := dc.singletones[name]; exists {
		return true
	}
	return false
}

func (dc *container) Lock() error {
	if dc.locked || dc.err != nil {
		return dc.err
	}
	dc.locked = true
	return nil
}

func (dc *container) Check(ctx context.Context) error {
	if err := dc.Lock(); err != nil {
		return err
	}
	for name, constructor := range dc.constructors {
		if _, err := constructor(ctx, dc); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	return nil
}

func (dc *container) Get(ctx context.Context, name string) (interface{}, error) {
	var res interface{}
	err := dc.Provide(ctx, name, &res)
	return res, err
}

func (dc *container) MustGet(ctx context.Context, name string) interface{} {
	res, err := dc.Get(ctx, name)
	if err != nil {
		panic(err)
	}
	return res
}

func (dc *container) Provide(ctx context.Context, name string, dst interface{}) error {
	if !dc.locked {
		return errors.New("container should be locked")
	}
	return dc.provide(ctx, name, reflect.ValueOf(dst).Elem())
}

func (dc *container) MustProvide(ctx context.Context, name string, dst interface{}) {
	if err := dc.Provide(ctx, name, dst); err != nil {
		panic(err)
	}
}

func (dc *container) MustProvideHTTPHandler(ctx context.Context, constructor interface{}) http.HandlerFunc {
	val := reflect.ValueOf(constructor)
	t := val.Type()
	if t.Kind() != reflect.Func {
		panic(fmt.Errorf("constructor should be an function, but got %T", constructor))
	}
	if t.NumOut() != 1 || t.Out(0).PkgPath() != "net/http" || t.Out(0).Name() != "HandlerFunc" {
		panic(fmt.Errorf(
			"constructor should return http.HandlerFunc, but got %T (%s.%s)",
			constructor, t.Out(0).PkgPath(), t.Out(0).Name()))
	}
	args := make([]reflect.Value, t.NumIn())
	for i := 0; i < t.NumIn(); i++ {
		arg := t.In(i)
		if arg.PkgPath() == "context" && arg.Name() == "Context" {
			args[i] = reflect.ValueOf(ctx)
		} else {
			res := reflect.New(arg).Elem()
			v := res
			if v.Kind() == reflect.Ptr {
				v = v.Elem()
			}
			if err := dc.provideStruct(ctx, v); err != nil {
				panic(err)
			}
			args[i] = res
		}
	}
	res := val.Call(args)
	return res[0].Interface().(http.HandlerFunc)
}

func (dc *container) provide(ctx context.Context, name string, dstValue reflect.Value) error {
	if service, ok := dc.singletones[name]; ok {
		dstValue.Set(service)
		return nil
	}
	constructor, ok := dc.constructors[name]
	if !ok {
		return fmt.Errorf("service %s not found", name)
	}
	// TODO: detect cycle dependency
	service, err := constructor(ctx, dc)
	if err != nil {
		return err
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
