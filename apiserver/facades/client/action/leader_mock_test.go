// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/juju/juju/core/leadership (interfaces: Reader)

// Package action is a generated GoMock package.
package action

import (
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"
)

// MockReader is a mock of Reader interface.
type MockReader struct {
	ctrl     *gomock.Controller
	recorder *MockReaderMockRecorder
}

// MockReaderMockRecorder is the mock recorder for MockReader.
type MockReaderMockRecorder struct {
	mock *MockReader
}

// NewMockReader creates a new mock instance.
func NewMockReader(ctrl *gomock.Controller) *MockReader {
	mock := &MockReader{ctrl: ctrl}
	mock.recorder = &MockReaderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockReader) EXPECT() *MockReaderMockRecorder {
	return m.recorder
}

// Leaders mocks base method.
func (m *MockReader) Leaders() (map[string]string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Leaders")
	ret0, _ := ret[0].(map[string]string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Leaders indicates an expected call of Leaders.
func (mr *MockReaderMockRecorder) Leaders() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Leaders", reflect.TypeOf((*MockReader)(nil).Leaders))
}
