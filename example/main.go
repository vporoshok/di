package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/vporoshok/di"
)

type User struct {
	Email string
	Name  string
}

type UserRepository struct {
	data *sync.Map
}

func NewUserRepository(_ context.Context, _ di.Container) (interface{}, error) {
	return &UserRepository{new(sync.Map)}, nil
}

func (ur *UserRepository) Insert(user *User) error {
	_, exists := ur.data.LoadOrStore(user.Email, user)
	if exists {
		return fmt.Errorf("user with email %s already exists", user.Email)
	}
	return nil
}

func (ur *UserRepository) Get(email string) (*User, error) {
	res, ok := ur.data.Load(email)
	if !ok {
		return nil, fmt.Errorf("user with email %s not found", email)
	}
	return res.(*User), nil
}

type CreateUser struct {
	Rep *UserRepository `di:"user repository"`
}

func (act *CreateUser) Do(email, name string) error {
	user := &User{
		strings.ToLower(email),
		name,
	}
	return act.Rep.Insert(user)
}

type GetUser struct {
	Rep interface {
		Get(string) (*User, error)
	} `di:"user repository"`
}

func main() {
	ts := time.Now()
	dc := di.NewContainer()
	dc.RegisterFunc("user repository", NewUserRepository)
	dc.RegisterStruct("create user", CreateUser{})
	dc.RegisterStruct("get user", GetUser{})
	if err := dc.Check(context.Background()); err != nil {
		log.Fatal(err)
	}
	log.Printf("dc created in %.2fms", float64(time.Since(ts))/float64(time.Millisecond))
	var act interface {
		Do(email, name string) error
	}
	if err := dc.Get(context.Background(), "create user", &act); err != nil {
		log.Fatal(err)
	}
	if err := act.Do("me@example.com", "test"); err != nil {
		log.Fatal(err)
	}
	log.Print("users added")
	if err := dc.Get(context.Background(), "create user", &act); err != nil {
		log.Fatal(err)
	}
	if err := act.Do("me@example.com", "test"); err != nil {
		log.Fatal(err)
	}
	log.Print("users added")
	t := &GetUser{}
	if err := dc.Get(context.Background(), "get user", &t); err != nil {
		log.Fatal(err)
	}
}
