package tokens

type Accumulator interface {
	Init(userID, sessionID string) error
	Summary(userID, sessionID string) (any, error)
	Initialize(userID, sessionID string) (string, error)
	Add(userID, sessionID string, n int)
	Cleanup(userID, sessionID string) error
}
