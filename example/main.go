package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

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
	rep *UserRepository
}

func NewCreateUser(ctx context.Context, dc di.Container) (interface{}, error) {
	var rep *UserRepository
	if err := dc.Get(ctx, "user repository", &rep); err != nil {
		return nil, err
	}
	return &CreateUser{rep}, nil
}

func (act *CreateUser) Do(email, name string) error {
	user := &User{
		strings.ToLower(email),
		name,
	}
	return act.rep.Insert(user)
}

func main() {
	dc := di.NewContainer()
	if err := dc.RegisterFunc("user repository", NewUserRepository); err != nil {
		log.Fatal(err)
	}
	if err := dc.RegisterFunc("create user", NewCreateUser); err != nil {
		log.Fatal(err)
	}
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
}
