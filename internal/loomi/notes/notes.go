package notes

type Note struct {
	ID         string
	UserID     string
	SessionID  string
	Action     string
	Name       string
	Title      string
	Content    string
	SelectFlag int
}

type Service interface {
	Create(userID, sessionID, action, name, title, content string, selectFlag int) error
	GetByAction(userID, sessionID, action string) ([]Note, error)
}
