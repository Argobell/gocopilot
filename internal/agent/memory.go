package agent

import (
	"sync"

	"github.com/openai/openai-go/v3"
)

const DefaultMemoryCapacity = 40

type Memory struct {
	mu         sync.RWMutex
	system     []openai.ChatCompletionMessageParamUnion
	history    []openai.ChatCompletionMessageParamUnion
	maxHistory int
}

func NewMemory(maxHistory int) *Memory {
	m := &Memory{}
	if maxHistory > 0 {
		m.maxHistory = maxHistory
	}
	return m
}

func (m *Memory) SetSystemMessages(messages ...openai.ChatCompletionMessageParamUnion) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(messages) == 0 {
		m.system = nil
		return
	}

	m.system = make([]openai.ChatCompletionMessageParamUnion, len(messages))
	copy(m.system, messages)
}

func (m *Memory) Append(message openai.ChatCompletionMessageParamUnion) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.history = append(m.history, message)
	m.trimLocked()
}

func (m *Memory) AppendMany(messages []openai.ChatCompletionMessageParamUnion) {
	if len(messages) == 0 {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.history = append(m.history, messages...)
	m.trimLocked()
}

func (m *Memory) TrimTo(maxHistory int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.maxHistory = maxHistory
	m.trimLocked()
}

func (m *Memory) ResetHistory() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.history = nil
}

func (m *Memory) Context() []openai.ChatCompletionMessageParamUnion {
	m.mu.RLock()
	defer m.mu.RUnlock()

	total := len(m.system) + len(m.history)
	if total == 0 {
		return nil
	}

	out := make([]openai.ChatCompletionMessageParamUnion, total)
	copy(out, m.system)
	copy(out[len(m.system):], m.history)
	return out
}

func (m *Memory) MessageCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.system) + len(m.history)
}

func (m *Memory) trimLocked() {
	if m.maxHistory <= 0 {
		return
	}

	if len(m.history) <= m.maxHistory {
		return
	}

	keep := m.history[len(m.history)-m.maxHistory:]
	m.history = append([]openai.ChatCompletionMessageParamUnion(nil), keep...)
}
