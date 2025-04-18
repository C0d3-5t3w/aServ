package storage

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	RoleAdmin  = "admin"
	RoleUser   = "user"
	MaxPerPage = 25
)

type User struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Password  string    `json:"password"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Category struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	CreatedBy   string    `json:"created_by"`
}

type Tag struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	CreatedBy string    `json:"created_by"`
}

type Item struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Price       float64   `json:"price"`
	CategoryID  string    `json:"category_id"`
	Tags        []string  `json:"tags"`
	ImageURL    string    `json:"image_url"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	CreatedBy   string    `json:"created_by"`
}

type AuditLog struct {
	ID        string    `json:"id"`
	Action    string    `json:"action"`
	Entity    string    `json:"entity"`
	EntityID  string    `json:"entity_id"`
	UserID    string    `json:"user_id"`
	Timestamp time.Time `json:"timestamp"`
	Details   string    `json:"details"`
}

type Analytics struct {
	TotalUsers        int       `json:"total_users"`
	TotalItems        int       `json:"total_items"`
	TotalCategories   int       `json:"total_categories"`
	TotalTags         int       `json:"total_tags"`
	PopularCategories []string  `json:"popular_categories"`
	PopularTags       []string  `json:"popular_tags"`
	RecentActivities  []string  `json:"recent_activities"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type SearchResults struct {
	Users      []User     `json:"users"`
	Items      []Item     `json:"items"`
	Categories []Category `json:"categories"`
	Tags       []Tag      `json:"tags"`
}

type StorageData struct {
	Users      map[string]User     `json:"users"`
	Items      map[string]Item     `json:"items"`
	Categories map[string]Category `json:"categories"`
	Tags       map[string]Tag      `json:"tags"`
	AuditLogs  map[string]AuditLog `json:"audit_logs"`
	Analytics  Analytics           `json:"analytics"`
}

type Storage struct {
	data     StorageData
	filePath string
	mu       sync.RWMutex
}

type PaginationParams struct {
	Page     int
	PerPage  int
	SortBy   string
	SortDesc bool
	Filter   map[string]string
}

type PaginatedResult struct {
	Data       interface{} `json:"data"`
	Page       int         `json:"page"`
	PerPage    int         `json:"per_page"`
	TotalItems int         `json:"total_items"`
	TotalPages int         `json:"total_pages"`
}

func NewStorage() *Storage {
	s := &Storage{
		filePath: "./pkg/storage/storage.json",
		data: StorageData{
			Users:      make(map[string]User),
			Items:      make(map[string]Item),
			Categories: make(map[string]Category),
			Tags:       make(map[string]Tag),
			AuditLogs:  make(map[string]AuditLog),
			Analytics: Analytics{
				UpdatedAt: time.Now(),
			},
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

	user.UpdatedAt = time.Now()
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

func (s *Storage) UpdateUserRole(id string, role string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, exists := s.data.Users[id]
	if !exists {
		return errors.New("user not found")
	}

	if role != RoleAdmin && role != RoleUser {
		return errors.New("invalid role")
	}

	user.Role = role
	user.UpdatedAt = time.Now()
	s.data.Users[id] = user
	return s.saveData()
}

func (s *Storage) SearchUsers(query string) []User {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := []User{}
	for _, user := range s.data.Users {
		if containsInsensitive(user.Username, query) ||
			containsInsensitive(user.Email, query) {
			userCopy := user
			userCopy.Password = ""
			result = append(result, userCopy)
		}
	}
	return result
}

func (s *Storage) GetUsersPaginated(params PaginationParams) PaginatedResult {
	s.mu.RLock()
	defer s.mu.RUnlock()

	users := s.getFilteredUsers(params.Filter)
	total := len(users)

	if params.PerPage <= 0 {
		params.PerPage = MaxPerPage
	}
	if params.Page <= 0 {
		params.Page = 1
	}

	start := (params.Page - 1) * params.PerPage
	end := start + params.PerPage
	if end > total {
		end = total
	}
	if start > total {
		start = total
	}

	totalPages := (total + params.PerPage - 1) / params.PerPage
	if totalPages < 1 {
		totalPages = 1
	}

	sortUsers(users, params.SortBy, params.SortDesc)

	pagedUsers := []User{}
	if start < total {
		pagedUsers = users[start:end]
	}

	for i := range pagedUsers {
		pagedUsers[i].Password = ""
	}

	return PaginatedResult{
		Data:       pagedUsers,
		Page:       params.Page,
		PerPage:    params.PerPage,
		TotalItems: total,
		TotalPages: totalPages,
	}
}

func (s *Storage) getFilteredUsers(filters map[string]string) []User {
	users := make([]User, 0, len(s.data.Users))

	for _, user := range s.data.Users {
		match := true

		for key, value := range filters {
			switch key {
			case "role":
				if user.Role != value {
					match = false
				}
			}
		}

		if match {
			users = append(users, user)
		}
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

	item.UpdatedAt = time.Now()
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

func (s *Storage) SearchItems(query string) []Item {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := []Item{}
	for _, item := range s.data.Items {
		if containsInsensitive(item.Name, query) ||
			containsInsensitive(item.Description, query) {
			result = append(result, item)
		}
	}
	return result
}

func (s *Storage) GetItemsByCategory(categoryID string) []Item {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := []Item{}
	for _, item := range s.data.Items {
		if item.CategoryID == categoryID {
			result = append(result, item)
		}
	}
	return result
}

func (s *Storage) GetItemsByTag(tagID string) []Item {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := []Item{}
	for _, item := range s.data.Items {
		for _, tag := range item.Tags {
			if tag == tagID {
				result = append(result, item)
				break
			}
		}
	}
	return result
}

func (s *Storage) GetItemsPaginated(params PaginationParams) PaginatedResult {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := s.getFilteredItems(params.Filter)
	total := len(items)

	if params.PerPage <= 0 {
		params.PerPage = MaxPerPage
	}
	if params.Page <= 0 {
		params.Page = 1
	}

	start := (params.Page - 1) * params.PerPage
	end := start + params.PerPage
	if end > total {
		end = total
	}
	if start > total {
		start = total
	}

	totalPages := (total + params.PerPage - 1) / params.PerPage
	if totalPages < 1 {
		totalPages = 1
	}

	sortItems(items, params.SortBy, params.SortDesc)

	pagedItems := []Item{}
	if start < total {
		pagedItems = items[start:end]
	}

	return PaginatedResult{
		Data:       pagedItems,
		Page:       params.Page,
		PerPage:    params.PerPage,
		TotalItems: total,
		TotalPages: totalPages,
	}
}

func (s *Storage) getFilteredItems(filters map[string]string) []Item {
	items := make([]Item, 0, len(s.data.Items))

	for _, item := range s.data.Items {
		match := true

		for key, value := range filters {
			switch key {
			case "category_id":
				if item.CategoryID != value {
					match = false
				}
			case "tag":
				found := false
				for _, tag := range item.Tags {
					if tag == value {
						found = true
						break
					}
				}
				if !found {
					match = false
				}
			}
		}

		if match {
			items = append(items, item)
		}
	}

	return items
}

func (s *Storage) CreateCategory(category Category) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data.Categories[category.ID] = category
	return s.saveData()
}

func (s *Storage) GetCategory(id string) (Category, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	category, exists := s.data.Categories[id]
	if !exists {
		return Category{}, errors.New("category not found")
	}
	return category, nil
}

func (s *Storage) UpdateCategory(category Category) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.data.Categories[category.ID]; !exists {
		return errors.New("category not found")
	}

	s.data.Categories[category.ID] = category
	return s.saveData()
}

func (s *Storage) DeleteCategory(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.data.Categories[id]; !exists {
		return errors.New("category not found")
	}

	delete(s.data.Categories, id)
	return s.saveData()
}

func (s *Storage) ListCategories() []Category {
	s.mu.RLock()
	defer s.mu.RUnlock()

	categories := make([]Category, 0, len(s.data.Categories))
	for _, category := range s.data.Categories {
		categories = append(categories, category)
	}
	return categories
}

func (s *Storage) CreateTag(tag Tag) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data.Tags[tag.ID] = tag
	return s.saveData()
}

func (s *Storage) GetTag(id string) (Tag, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tag, exists := s.data.Tags[id]
	if !exists {
		return Tag{}, errors.New("tag not found")
	}
	return tag, nil
}

func (s *Storage) DeleteTag(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.data.Tags[id]; !exists {
		return errors.New("tag not found")
	}

	delete(s.data.Tags, id)
	return s.saveData()
}

func (s *Storage) ListTags() []Tag {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tags := make([]Tag, 0, len(s.data.Tags))
	for _, tag := range s.data.Tags {
		tags = append(tags, tag)
	}
	return tags
}

func (s *Storage) CreateAuditLog(log AuditLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data.AuditLogs[log.ID] = log
	return s.saveData()
}

func (s *Storage) GetAuditLogs(limit int) []AuditLog {
	s.mu.RLock()
	defer s.mu.RUnlock()

	logs := make([]AuditLog, 0, len(s.data.AuditLogs))
	for _, log := range s.data.AuditLogs {
		logs = append(logs, log)
	}

	sortAuditLogs(logs)

	if limit > 0 && limit < len(logs) {
		logs = logs[:limit]
	}

	return logs
}

func (s *Storage) UpdateAnalytics() {
	s.mu.Lock()
	defer s.mu.Unlock()

	analytics := Analytics{
		TotalUsers:        len(s.data.Users),
		TotalItems:        len(s.data.Items),
		TotalCategories:   len(s.data.Categories),
		TotalTags:         len(s.data.Tags),
		PopularCategories: s.getPopularCategories(5),
		PopularTags:       s.getPopularTags(5),
		RecentActivities:  s.getRecentActivities(10),
		UpdatedAt:         time.Now(),
	}

	s.data.Analytics = analytics
	s.saveData()
}

func (s *Storage) GetAnalytics() Analytics {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data.Analytics
}

func (s *Storage) getPopularCategories(limit int) []string {
	categoryCounts := make(map[string]int)
	for _, item := range s.data.Items {
		categoryCounts[item.CategoryID]++
	}

	counts := []categoryCount{}
	for id, count := range categoryCounts {
		counts = append(counts, categoryCount{ID: id, Count: count})
	}

	sortByCounts(counts)

	result := []string{}
	for i := 0; i < len(counts) && i < limit; i++ {
		if category, ok := s.data.Categories[counts[i].ID]; ok {
			result = append(result, category.Name)
		}
	}

	return result
}

func (s *Storage) getPopularTags(limit int) []string {
	tagCounts := make(map[string]int)
	for _, item := range s.data.Items {
		for _, tagID := range item.Tags {
			tagCounts[tagID]++
		}
	}

	counts := []tagCount{}
	for id, count := range tagCounts {
		counts = append(counts, tagCount{ID: id, Count: count})
	}

	sortByCounts(counts)

	result := []string{}
	for i := 0; i < len(counts) && i < limit; i++ {
		if tag, ok := s.data.Tags[counts[i].ID]; ok {
			result = append(result, tag.Name)
		}
	}

	return result
}

func (s *Storage) getRecentActivities(limit int) []string {
	logs := s.GetAuditLogs(limit)

	activities := make([]string, len(logs))
	for i, log := range logs {
		var username string
		if user, ok := s.data.Users[log.UserID]; ok {
			username = user.Username
		} else {
			username = "Unknown"
		}

		activities[i] = formatActivity(log, username)
	}

	return activities
}

func containsInsensitive(s, substr string) bool {
	s, substr = strings.ToLower(s), strings.ToLower(substr)
	return strings.Contains(s, substr)
}

func sortUsers(users []User, sortBy string, desc bool) {
}

func sortItems(items []Item, sortBy string, desc bool) {
}

func sortAuditLogs(logs []AuditLog) {
}

func sortByCounts[T interface {
	GetID() string
	GetCount() int
}](items []T) {
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			if items[i].GetCount() < items[j].GetCount() {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
}

type categoryCount struct {
	ID    string
	Count int
}

type tagCount struct {
	ID    string
	Count int
}

func (c categoryCount) GetID() string { return c.ID }
func (c categoryCount) GetCount() int { return c.Count }

func (t tagCount) GetID() string { return t.ID }
func (t tagCount) GetCount() int { return t.Count }

func formatActivity(log AuditLog, username string) string {
	return "Activity message format"
}
