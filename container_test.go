package di_test

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/vporoshok/di"
)

func ExampleContainer() {
	type A struct {
		Foo string `di:"foo"`
	}
	type B struct{}
	type Comb struct {
		A A  `di:"a"`
		B *B `di:"b"`
	}
	dc := di.NewContainer()
	dc.RegisterInstance("foo", "bar")
	dc.RegisterStruct("a", A{})
	dc.RegisterFunc("b", func(_ context.Context, x B) (*B, error) {
		return &x, nil
	}, di.Singletone())
	ctx := context.Background()
	if err := dc.Check(ctx); err != nil {
		log.Fatal(err)
	}
	var c Comb
	if err := dc.ProvideStruct(ctx, &c); err != nil {
		log.Fatal(err)
	}
	fmt.Println(c.A.Foo)
	// Output: bar
}

func ExampleContainer_Get() {
	dc := di.NewContainer()
	dc.RegisterInstance("log", log.New(os.Stdout, "", 0))
	if err := dc.Lock(); err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	logger := dc.MustGet(ctx, "log").(*log.Logger)
	logger.Print("test")
	// Output: test
}
