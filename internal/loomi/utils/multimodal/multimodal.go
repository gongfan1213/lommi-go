package multimodal

import "context"

type Processor interface {
	Enabled() bool
	ShouldUse(ctx context.Context, userID, sessionID, prompt string) (bool, error)
	Process(ctx context.Context, userID, sessionID, prompt, agentName string) (results any, err error)
	SaveAsNotes(ctx context.Context, results any, userID, sessionID string) (saved int, err error)
}
