package model

type UserCreate struct {
	FullName    string
	Email       string
	PhoneNumber string
	Photo       string
}

type User struct {
	ID        int64
	PushToken string
	Notify    bool
	UserCreate
}

type UserSearchFilter struct {
	Query string
	Limit int
	Page  int
}
