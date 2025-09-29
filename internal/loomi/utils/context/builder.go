package contextx

import (
	"context"
	"fmt"
	"strings"
)

// BuildConciergeContext builds the user prompt with conversation context and selections
func BuildConciergeContext(ctx context.Context, userID, sessionID, currentUserMessage string, userSelections []string) (string, error) {
	parts := []string{}
	if len(userSelections) > 0 {
		parts = append(parts, fmt.Sprintf("[用户选择]\n%s", strings.Join(userSelections, "\n")))
	}
	if trimmed := strings.TrimSpace(currentUserMessage); trimmed != "" {
		parts = append(parts, fmt.Sprintf("[当前需求]\n%s", trimmed))
	}
	// TODO: attach conversation history and notes summary when services are wired
	return strings.Join(parts, "\n\n"), nil
}
