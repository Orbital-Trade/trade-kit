package ops

// mockCaller is a test double for the Caller interface.
// Responses and errors are queued per method; each Call() drains the next item.

import (
	"encoding/json"
	"errors"
)

type mockCaller struct {
	paper     bool
	account   string
	responses map[string][]json.RawMessage
	errs      map[string][]error
	counts    map[string]int // how many times each method was called
}

func newMock(paper bool) *mockCaller {
	return &mockCaller{
		paper:     paper,
		account:   "TEST-001",
		responses: make(map[string][]json.RawMessage),
		errs:      make(map[string][]error),
		counts:    make(map[string]int),
	}
}

// on queues a response (and optional error) for the next call to method.
// Call repeatedly to set up a sequence: first call gets [0], second gets [1], etc.
func (m *mockCaller) on(method string, data interface{}, err error) *mockCaller {
	var raw json.RawMessage
	if data != nil {
		b, e := json.Marshal(data)
		if e != nil {
			panic("mockCaller.on: marshal failed: " + e.Error())
		}
		raw = b
	}
	m.responses[method] = append(m.responses[method], raw)
	m.errs[method] = append(m.errs[method], err)
	return m
}

// onErr queues an error (nil data) for the next call to method.
func (m *mockCaller) onErr(method, msg string) *mockCaller {
	return m.on(method, nil, errors.New(msg))
}

func (m *mockCaller) Call(method string, _ interface{}) (json.RawMessage, error) {
	idx := m.counts[method]
	m.counts[method]++

	var data json.RawMessage
	var err error
	if idx < len(m.responses[method]) {
		data = m.responses[method][idx]
	}
	if idx < len(m.errs[method]) {
		err = m.errs[method][idx]
	}
	return data, err
}

func (m *mockCaller) Account() string { return m.account }
func (m *mockCaller) IsPaper() bool   { return m.paper }
func (m *mockCaller) ResolveFuturesContract(symbol string) (string, error) {
	return symbol + "2506", nil
}

// called returns how many times method was invoked.
func (m *mockCaller) called(method string) int { return m.counts[method] }
