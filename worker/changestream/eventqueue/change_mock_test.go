// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/juju/juju/core/changestream (interfaces: ChangeEvent)
//
// Generated by this command:
//
//	mockgen -package eventqueue -destination change_mock_test.go github.com/juju/juju/core/changestream ChangeEvent
//

// Package eventqueue is a generated GoMock package.
package eventqueue

import (
	reflect "reflect"

	changestream "github.com/juju/juju/core/changestream"
	gomock "go.uber.org/mock/gomock"
)

// MockChangeEvent is a mock of ChangeEvent interface.
type MockChangeEvent struct {
	ctrl     *gomock.Controller
	recorder *MockChangeEventMockRecorder
}

// MockChangeEventMockRecorder is the mock recorder for MockChangeEvent.
type MockChangeEventMockRecorder struct {
	mock *MockChangeEvent
}

// NewMockChangeEvent creates a new mock instance.
func NewMockChangeEvent(ctrl *gomock.Controller) *MockChangeEvent {
	mock := &MockChangeEvent{ctrl: ctrl}
	mock.recorder = &MockChangeEventMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockChangeEvent) EXPECT() *MockChangeEventMockRecorder {
	return m.recorder
}

// ChangedUUID mocks base method.
func (m *MockChangeEvent) ChangedUUID() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ChangedUUID")
	ret0, _ := ret[0].(string)
	return ret0
}

// ChangedUUID indicates an expected call of ChangedUUID.
func (mr *MockChangeEventMockRecorder) ChangedUUID() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ChangedUUID", reflect.TypeOf((*MockChangeEvent)(nil).ChangedUUID))
}

// Namespace mocks base method.
func (m *MockChangeEvent) Namespace() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Namespace")
	ret0, _ := ret[0].(string)
	return ret0
}

// Namespace indicates an expected call of Namespace.
func (mr *MockChangeEventMockRecorder) Namespace() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Namespace", reflect.TypeOf((*MockChangeEvent)(nil).Namespace))
}

// Type mocks base method.
func (m *MockChangeEvent) Type() changestream.ChangeType {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Type")
	ret0, _ := ret[0].(changestream.ChangeType)
	return ret0
}

// Type indicates an expected call of Type.
func (mr *MockChangeEventMockRecorder) Type() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Type", reflect.TypeOf((*MockChangeEvent)(nil).Type))
}
