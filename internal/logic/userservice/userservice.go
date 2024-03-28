package userservice

type UserService interface {
	IsAdmin(userID int64) bool
	IsUserAllowed(userID, chatID int64) bool
}
