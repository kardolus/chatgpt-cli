// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/kardolus/chatgpt-cli/history (interfaces: HistoryStore)

// Package history_test is a generated GoMock package.
package history_test

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	types "github.com/kardolus/chatgpt-cli/types"
)

// MockHistoryStore is a mock of HistoryStore interface.
type MockHistoryStore struct {
	ctrl     *gomock.Controller
	recorder *MockHistoryStoreMockRecorder
}

// MockHistoryStoreMockRecorder is the mock recorder for MockHistoryStore.
type MockHistoryStoreMockRecorder struct {
	mock *MockHistoryStore
}

// NewMockHistoryStore creates a new mock instance.
func NewMockHistoryStore(ctrl *gomock.Controller) *MockHistoryStore {
	mock := &MockHistoryStore{ctrl: ctrl}
	mock.recorder = &MockHistoryStoreMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockHistoryStore) EXPECT() *MockHistoryStoreMockRecorder {
	return m.recorder
}

// GetThread mocks base method.
func (m *MockHistoryStore) GetThread() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetThread")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetThread indicates an expected call of GetThread.
func (mr *MockHistoryStoreMockRecorder) GetThread() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetThread", reflect.TypeOf((*MockHistoryStore)(nil).GetThread))
}

// Read mocks base method.
func (m *MockHistoryStore) Read() ([]types.History, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Read")
	ret0, _ := ret[0].([]types.History)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Read indicates an expected call of Read.
func (mr *MockHistoryStoreMockRecorder) Read() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Read", reflect.TypeOf((*MockHistoryStore)(nil).Read))
}

// ReadThread mocks base method.
func (m *MockHistoryStore) ReadThread(arg0 string) ([]types.History, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReadThread", arg0)
	ret0, _ := ret[0].([]types.History)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ReadThread indicates an expected call of ReadThread.
func (mr *MockHistoryStoreMockRecorder) ReadThread(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReadThread", reflect.TypeOf((*MockHistoryStore)(nil).ReadThread), arg0)
}

// SetThread mocks base method.
func (m *MockHistoryStore) SetThread(arg0 string) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetThread", arg0)
}

// SetThread indicates an expected call of SetThread.
func (mr *MockHistoryStoreMockRecorder) SetThread(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetThread", reflect.TypeOf((*MockHistoryStore)(nil).SetThread), arg0)
}

// Write mocks base method.
func (m *MockHistoryStore) Write(arg0 []types.History) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Write", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Write indicates an expected call of Write.
func (mr *MockHistoryStoreMockRecorder) Write(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Write", reflect.TypeOf((*MockHistoryStore)(nil).Write), arg0)
}