package ops

import (
	"encoding/json"
	"fmt"
)

type mockCaller struct {
	responses map[string]json.RawMessage
	paper     bool
	accountID string
	conids    map[string]int
}

func newMock(paper bool) *mockCaller {
	return &mockCaller{
		responses: make(map[string]json.RawMessage),
		paper:     paper,
		accountID: "MOCK_ACCT",
		conids:    make(map[string]int),
	}
}

func (m *mockCaller) setResponse(path string, v interface{}) {
	b, _ := json.Marshal(v)
	m.responses[path] = b
}

func (m *mockCaller) Get(path string, _ map[string]string) (json.RawMessage, error) {
	if data, ok := m.responses[path]; ok {
		return data, nil
	}
	return nil, fmt.Errorf("mock: no response for GET %s", path)
}

func (m *mockCaller) Post(path string, _ interface{}) (json.RawMessage, error) {
	if data, ok := m.responses[path]; ok {
		return data, nil
	}
	return nil, fmt.Errorf("mock: no response for POST %s", path)
}

func (m *mockCaller) Delete(path string, _ map[string]string) (json.RawMessage, error) {
	if data, ok := m.responses[path]; ok {
		return data, nil
	}
	return nil, fmt.Errorf("mock: no response for DELETE %s", path)
}

func (m *mockCaller) IsPaper() bool      { return m.paper }
func (m *mockCaller) AccountID() string   { return m.accountID }

func (m *mockCaller) ResolveConID(symbol string) (int, error) {
	if id, ok := m.conids[symbol]; ok {
		return id, nil
	}
	return 0, fmt.Errorf("mock: no conid for %s", symbol)
}
