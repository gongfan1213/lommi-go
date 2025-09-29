package contextx

import (
	"fmt"
	"strings"
	"sync"
)

type inmem struct {
	mu sync.Mutex
	// key: user:session => sequential id per action
	counters map[string]map[string]int
	// simple history buffer
	history map[string][]string
	// notes index: action -> []string
	notes map[string]map[string][]string
}

func NewInmem() Manager {
	return &inmem{counters: map[string]map[string]int{}, history: map[string][]string{}, notes: map[string]map[string][]string{}}
}

func (i *inmem) Get(userID, sessionID string) (*State, error) {
	return &State{UserID: userID, SessionID: sessionID}, nil
}
func (i *inmem) Create(userID, sessionID, initialQuery string) (*State, error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	key := fmt.Sprintf("%s:%s", userID, sessionID)
	i.history[key] = append(i.history[key], "user:"+initialQuery)
	return &State{UserID: userID, SessionID: sessionID}, nil
}
func (i *inmem) UpdateOrchestratorCallResponse(userID, sessionID string, callIndex int, responseContent string, observeThinkAction any) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	key := fmt.Sprintf("%s:%s", userID, sessionID)
	i.history[key] = append(i.history[key], "assistant:"+responseContent)
	return nil
}
func (i *inmem) NextActionID(userID, sessionID, action string) (int, error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	key := fmt.Sprintf("%s:%s", userID, sessionID)
	if _, ok := i.counters[key]; !ok {
		i.counters[key] = map[string]int{}
	}
	i.counters[key][action]++
	return i.counters[key][action], nil
}

func (i *inmem) FormatContextForPrompt(userID, sessionID, agentName string, includeHistory, includeNotes, includeSelections bool, selections []string, includeSystem, includeDebug bool) (string, error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	parts := []string{}
	key := fmt.Sprintf("%s:%s", userID, sessionID)
	if includeHistory {
		hist := i.history[key]
		if len(hist) > 0 {
			// keep last 6 messages
			if len(hist) > 6 {
				hist = hist[len(hist)-6:]
			}
			parts = append(parts, "[对话片段]"+"\n"+strings.Join(hist, "\n"))
		}
	}
	if includeSelections && len(selections) > 0 {
		parts = append(parts, "[用户选择]"+"\n"+strings.Join(selections, "\n"))
	}
	if includeNotes {
		if idx, ok := i.notes[key]; ok {
			for action, arr := range idx {
				if len(arr) > 0 {
					parts = append(parts, fmt.Sprintf("[Notes:%s]\n%s", action, strings.Join(arr, "\n")))
				}
			}
		}
	}
	return strings.Join(parts, "\n\n"), nil
}
