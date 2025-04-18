package storage

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"sync"
	"time"
)

type User struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Password  string    `json:"password"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type Item struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Price       float64   `json:"price"`
	CreatedAt   time.Time `json:"created_at"`
	CreatedBy   string    `json:"created_by"`
}

type StorageData struct {
	Users map[string]User `json:"users"`
	Items map[string]Item `json:"items"`
}

type Storage struct {
	data     StorageData
	filePath string
	mu       sync.RWMutex
}

func NewStorage() *Storage {
	s := &Storage{
		filePath: "./pkg/storage/storage.json",
		data: StorageData{
			Users: make(map[string]User),
			Items: make(map[string]Item),
		},
	}
	s.loadData()
	return s
}

func (s *Storage) loadData() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := ioutil.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return s.saveData()
		}
		return err
	}

	return json.Unmarshal(data, &s.data)
}

func (s *Storage) saveData() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(s.filePath, data, 0644)
}

func (s *Storage) GetUser(id string) (User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.data.Users[id]
	if !exists {
		return User{}, errors.New("user not found")
	}
	return user, nil
}

func (s *Storage) GetUserByUsername(username string) (User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, user := range s.data.Users {
		if user.Username == username {
			return user, nil
		}
	}
	return User{}, errors.New("user not found")
}

func (s *Storage) CreateUser(user User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data.Users[user.ID] = user
	return s.saveData()
}

func (s *Storage) UpdateUser(user User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.data.Users[user.ID]; !exists {
		return errors.New("user not found")
	}

	s.data.Users[user.ID] = user
	return s.saveData()
}

func (s *Storage) DeleteUser(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.data.Users[id]; !exists {
		return errors.New("user not found")
	}

	delete(s.data.Users, id)
	return s.saveData()
}

func (s *Storage) ListUsers() []User {
	s.mu.RLock()
	defer s.mu.RUnlock()

	users := make([]User, 0, len(s.data.Users))
	for _, user := range s.data.Users {
		users = append(users, user)
	}
	return users
}

func (s *Storage) GetItem(id string) (Item, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, exists := s.data.Items[id]
	if !exists {
		return Item{}, errors.New("item not found")
	}
	return item, nil
}

func (s *Storage) CreateItem(item Item) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data.Items[item.ID] = item
	return s.saveData()
}

func (s *Storage) UpdateItem(item Item) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.data.Items[item.ID]; !exists {
		return errors.New("item not found")
	}

	s.data.Items[item.ID] = item
	return s.saveData()
}

func (s *Storage) DeleteItem(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.data.Items[id]; !exists {
		return errors.New("item not found")
	}

	delete(s.data.Items, id)
	return s.saveData()
}

func (s *Storage) ListItems() []Item {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]Item, 0, len(s.data.Items))
	for _, item := range s.data.Items {
		items = append(items, item)
	}
	return items
}
