package ops

import (
	"encoding/json"
	"fmt"
)

type mockCaller struct {
	responses map[string]json.RawMessage
	paper     bool
}

func newMock(paper bool) *mockCaller {
	return &mockCaller{responses: make(map[string]json.RawMessage), paper: paper}
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

func (m *mockCaller) Patch(path string, _ interface{}) (json.RawMessage, error) {
	if data, ok := m.responses[path]; ok {
		return data, nil
	}
	return nil, fmt.Errorf("mock: no response for PATCH %s", path)
}

func (m *mockCaller) DataGet(path string, _ map[string]string) (json.RawMessage, error) {
	if data, ok := m.responses[path]; ok {
		return data, nil
	}
	return nil, fmt.Errorf("mock: no response for DataGet %s", path)
}

func (m *mockCaller) IsPaper() bool { return m.paper }
