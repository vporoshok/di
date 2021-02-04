package di_test

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	dc.RegisterFunc("log", func() (*log.Logger, error) {
		return log.New(os.Stdout, "", 0), nil
	})
	if err := dc.Lock(); err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	logger := dc.MustGet(ctx, "log").(*log.Logger)
	logger.Print("test")
	// Output: test
}

func ExampleContainer_RegisterStruct() {
	dc := di.NewContainer()
	dc.RegisterInstance("log", log.New(os.Stdout, "", 0))
	dc.RegisterStruct("a", &struct {
		*log.Logger `di:"log"`
	}{})
	if err := dc.Lock(); err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	logger := dc.MustGet(ctx, "a").(interface{ Print(...interface{}) })
	logger.Print("test")
	// Output: test
}

func TestDIName(t *testing.T) {
	dc := di.NewContainer()
	buf := new(bytes.Buffer)
	dc.RegisterInstance("log", log.New(buf, "", 0))
	require.NoError(t, dc.Lock())
	ctx := context.Background()
	container := dc.MustGet(ctx, "di").(di.Container)
	logger := container.MustGet(ctx, "log").(*log.Logger)
	logger.Print("test")
	assert.Equal(t, "test\n", buf.String())
}

func TestMustProvideHTTPHandler(t *testing.T) {
	dc := di.NewContainer()
	buf := new(bytes.Buffer)
	dc.RegisterInstance("log", log.New(buf, "", 0))
	require.NoError(t, dc.Lock())
	ctx := context.Background()
	handler := dc.MustProvideHTTPHandler(ctx, func(dep struct {
		L *log.Logger `di:"log"`
	}) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			dep.L.Print(r.URL.Path)
			w.WriteHeader(http.StatusOK)
		}
	})
	assert.HTTPStatusCode(t, handler, http.MethodGet, "/foo", nil, http.StatusOK)
	assert.Equal(t, "/foo\n", buf.String())
}
