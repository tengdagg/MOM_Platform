package biz

import "sync"

type SessionStreamPayload struct {
	SessionID uint
	RunID     string
	Type      string
	Content   string
	Event     AgentEvent
	Source    string
}

type SessionStreamBus struct {
	mu     sync.RWMutex
	nextID uint64
	subs   map[uint]map[uint64]chan SessionStreamPayload
}

func NewSessionStreamBus() *SessionStreamBus {
	return &SessionStreamBus{
		subs: make(map[uint]map[uint64]chan SessionStreamPayload),
	}
}

func (b *SessionStreamBus) Subscribe(userID uint) (uint64, <-chan SessionStreamPayload, func()) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.nextID++
	id := b.nextID
	ch := make(chan SessionStreamPayload, 128)
	if _, ok := b.subs[userID]; !ok {
		b.subs[userID] = make(map[uint64]chan SessionStreamPayload)
	}
	b.subs[userID][id] = ch

	unsubscribe := func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if userSubs, ok := b.subs[userID]; ok {
			if subCh, exists := userSubs[id]; exists {
				delete(userSubs, id)
				close(subCh)
			}
			if len(userSubs) == 0 {
				delete(b.subs, userID)
			}
		}
	}
	return id, ch, unsubscribe
}

func (b *SessionStreamBus) Publish(userID uint, payload SessionStreamPayload) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	userSubs := b.subs[userID]
	for _, ch := range userSubs {
		select {
		case ch <- payload:
		default:
		}
	}
}

var GlobalSessionStreamBus = NewSessionStreamBus()
