// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/juju/juju/core/watcher (interfaces: SecretTriggerWatcher)
//
// Generated by this command:
//
//	mockgen -package mocks -destination mocks/watcher_mock.go github.com/juju/juju/core/watcher SecretTriggerWatcher
//

// Package mocks is a generated GoMock package.
package mocks

import (
	reflect "reflect"

	watcher "github.com/juju/juju/core/watcher"
	gomock "go.uber.org/mock/gomock"
)

// MockSecretTriggerWatcher is a mock of SecretTriggerWatcher interface.
type MockSecretTriggerWatcher struct {
	ctrl     *gomock.Controller
	recorder *MockSecretTriggerWatcherMockRecorder
}

// MockSecretTriggerWatcherMockRecorder is the mock recorder for MockSecretTriggerWatcher.
type MockSecretTriggerWatcherMockRecorder struct {
	mock *MockSecretTriggerWatcher
}

// NewMockSecretTriggerWatcher creates a new mock instance.
func NewMockSecretTriggerWatcher(ctrl *gomock.Controller) *MockSecretTriggerWatcher {
	mock := &MockSecretTriggerWatcher{ctrl: ctrl}
	mock.recorder = &MockSecretTriggerWatcherMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockSecretTriggerWatcher) EXPECT() *MockSecretTriggerWatcherMockRecorder {
	return m.recorder
}

// Changes mocks base method.
func (m *MockSecretTriggerWatcher) Changes() watcher.SecretTriggerChannel {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Changes")
	ret0, _ := ret[0].(watcher.SecretTriggerChannel)
	return ret0
}

// Changes indicates an expected call of Changes.
func (mr *MockSecretTriggerWatcherMockRecorder) Changes() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Changes", reflect.TypeOf((*MockSecretTriggerWatcher)(nil).Changes))
}

// Kill mocks base method.
func (m *MockSecretTriggerWatcher) Kill() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Kill")
}

// Kill indicates an expected call of Kill.
func (mr *MockSecretTriggerWatcherMockRecorder) Kill() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Kill", reflect.TypeOf((*MockSecretTriggerWatcher)(nil).Kill))
}

// Wait mocks base method.
func (m *MockSecretTriggerWatcher) Wait() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Wait")
	ret0, _ := ret[0].(error)
	return ret0
}

// Wait indicates an expected call of Wait.
func (mr *MockSecretTriggerWatcherMockRecorder) Wait() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Wait", reflect.TypeOf((*MockSecretTriggerWatcher)(nil).Wait))
}
