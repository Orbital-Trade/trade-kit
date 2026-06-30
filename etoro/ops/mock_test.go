package ops

import (
	"encoding/json"
	"fmt"
)

// mockCaller is a test double for ops.Caller.
// It returns pre-configured responses keyed by path.
type mockCaller struct {
	responses map[string]json.RawMessage
	paper     bool
}

func newMock(paper bool) *mockCaller {
	return &mockCaller{
		responses: make(map[string]json.RawMessage),
		paper:     paper,
	}
}

func (m *mockCaller) setResponse(path string, v interface{}) {
	b, _ := json.Marshal(v)
	m.responses[path] = b
}

func (m *mockCaller) Get(path string, query map[string]string) (json.RawMessage, error) {
	if data, ok := m.responses[path]; ok {
		return data, nil
	}
	return nil, fmt.Errorf("mock: no response for GET %s", path)
}

func (m *mockCaller) Post(path string, body interface{}) (json.RawMessage, error) {
	if data, ok := m.responses[path]; ok {
		return data, nil
	}
	return nil, fmt.Errorf("mock: no response for POST %s", path)
}

func (m *mockCaller) Put(path string, body interface{}) (json.RawMessage, error) {
	if data, ok := m.responses[path]; ok {
		return data, nil
	}
	return nil, fmt.Errorf("mock: no response for PUT %s", path)
}

func (m *mockCaller) Delete(path string, query map[string]string) (json.RawMessage, error) {
	if data, ok := m.responses[path]; ok {
		return data, nil
	}
	return nil, fmt.Errorf("mock: no response for DELETE %s", path)
}

func (m *mockCaller) Patch(path string, body interface{}) (json.RawMessage, error) {
	if data, ok := m.responses[path]; ok {
		return data, nil
	}
	return nil, fmt.Errorf("mock: no response for PATCH %s", path)
}

func (m *mockCaller) IsPaper() bool { return m.paper }
