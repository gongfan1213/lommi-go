package notes

// InmemService provides an in-memory implementation of the notes service
type InmemService struct {
	notes map[string][]Note
}

// NewInmem creates a new in-memory notes service
func NewInmem() Service {
	return &InmemService{
		notes: make(map[string][]Note),
	}
}

func (s *InmemService) Create(userID, sessionID, action, name, title, content string, selectFlag int) error {
	key := userID + ":" + sessionID
	note := Note{
		ID:         name,
		UserID:     userID,
		SessionID:  sessionID,
		Action:     action,
		Name:       name,
		Title:      title,
		Content:    content,
		SelectFlag: selectFlag,
	}
	s.notes[key] = append(s.notes[key], note)
	return nil
}

func (s *InmemService) GetByAction(userID, sessionID, action string) ([]Note, error) {
	key := userID + ":" + sessionID
	allNotes := s.notes[key]
	var result []Note
	for _, note := range allNotes {
		if note.Action == action {
			result = append(result, note)
		}
	}
	return result, nil
}
