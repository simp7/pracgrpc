package main

import (
	"errors"
	"github.com/simp7/pracgrpc/model"
	"sync"
)

var (
	ErrAlreadyExists = errors.New("user already exist")
)

type UserStore interface {
	Save(user *model_v0_0_0_20240103072605_23fc1710dcc0.User) error
	Find(username string) (*model_v0_0_0_20240103072605_23fc1710dcc0.User, error)
}

type InMemoryUserStore struct {
	mutex sync.RWMutex
	users map[string]*model_v0_0_0_20240103072605_23fc1710dcc0.User
}

func (store *InMemoryUserStore) Save(user *model_v0_0_0_20240103072605_23fc1710dcc0.User) error {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	if store.users[user.Username] != nil {
		return ErrAlreadyExists
	}

	store.users[user.Username] = user.Clone()
	return nil
}

func (store *InMemoryUserStore) Find(username string) (*model_v0_0_0_20240103072605_23fc1710dcc0.User, error) {
	store.mutex.RLock()
	defer store.mutex.RUnlock()

	user := store.users[username]
	if user == nil {
		return nil, nil
	}

	return user.Clone(), nil
}

func NewInMemoryUserStore() *InMemoryUserStore {
	return &InMemoryUserStore{
		users: make(map[string]*model_v0_0_0_20240103072605_23fc1710dcc0.User),
	}
}
