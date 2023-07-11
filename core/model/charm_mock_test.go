// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/juju/charm/v8 (interfaces: CharmMeta)

// Package model_test is a generated GoMock package.
package model_test

import (
	reflect "reflect"

	charm "github.com/juju/charm/v8"
	gomock "go.uber.org/mock/gomock"
)

// MockCharmMeta is a mock of CharmMeta interface.
type MockCharmMeta struct {
	ctrl     *gomock.Controller
	recorder *MockCharmMetaMockRecorder
}

// MockCharmMetaMockRecorder is the mock recorder for MockCharmMeta.
type MockCharmMetaMockRecorder struct {
	mock *MockCharmMeta
}

// NewMockCharmMeta creates a new mock instance.
func NewMockCharmMeta(ctrl *gomock.Controller) *MockCharmMeta {
	mock := &MockCharmMeta{ctrl: ctrl}
	mock.recorder = &MockCharmMetaMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCharmMeta) EXPECT() *MockCharmMetaMockRecorder {
	return m.recorder
}

// Manifest mocks base method.
func (m *MockCharmMeta) Manifest() *charm.Manifest {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Manifest")
	ret0, _ := ret[0].(*charm.Manifest)
	return ret0
}

// Manifest indicates an expected call of Manifest.
func (mr *MockCharmMetaMockRecorder) Manifest() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Manifest", reflect.TypeOf((*MockCharmMeta)(nil).Manifest))
}

// Meta mocks base method.
func (m *MockCharmMeta) Meta() *charm.Meta {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Meta")
	ret0, _ := ret[0].(*charm.Meta)
	return ret0
}

// Meta indicates an expected call of Meta.
func (mr *MockCharmMetaMockRecorder) Meta() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Meta", reflect.TypeOf((*MockCharmMeta)(nil).Meta))
}
