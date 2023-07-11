// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/juju/juju/core/charm/repository (interfaces: CharmHubClient)

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	url "net/url"
	reflect "reflect"

	charm "github.com/juju/charm/v10"
	charmhub "github.com/juju/juju/charmhub"
	transport "github.com/juju/juju/charmhub/transport"
	gomock "go.uber.org/mock/gomock"
)

// MockCharmHubClient is a mock of CharmHubClient interface.
type MockCharmHubClient struct {
	ctrl     *gomock.Controller
	recorder *MockCharmHubClientMockRecorder
}

// MockCharmHubClientMockRecorder is the mock recorder for MockCharmHubClient.
type MockCharmHubClientMockRecorder struct {
	mock *MockCharmHubClient
}

// NewMockCharmHubClient creates a new mock instance.
func NewMockCharmHubClient(ctrl *gomock.Controller) *MockCharmHubClient {
	mock := &MockCharmHubClient{ctrl: ctrl}
	mock.recorder = &MockCharmHubClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCharmHubClient) EXPECT() *MockCharmHubClientMockRecorder {
	return m.recorder
}

// DownloadAndRead mocks base method.
func (m *MockCharmHubClient) DownloadAndRead(arg0 context.Context, arg1 *url.URL, arg2 string, arg3 ...charmhub.DownloadOption) (*charm.CharmArchive, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1, arg2}
	for _, a := range arg3 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "DownloadAndRead", varargs...)
	ret0, _ := ret[0].(*charm.CharmArchive)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DownloadAndRead indicates an expected call of DownloadAndRead.
func (mr *MockCharmHubClientMockRecorder) DownloadAndRead(arg0, arg1, arg2 interface{}, arg3 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1, arg2}, arg3...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DownloadAndRead", reflect.TypeOf((*MockCharmHubClient)(nil).DownloadAndRead), varargs...)
}

// Refresh mocks base method.
func (m *MockCharmHubClient) Refresh(arg0 context.Context, arg1 charmhub.RefreshConfig) ([]transport.RefreshResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Refresh", arg0, arg1)
	ret0, _ := ret[0].([]transport.RefreshResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Refresh indicates an expected call of Refresh.
func (mr *MockCharmHubClientMockRecorder) Refresh(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Refresh", reflect.TypeOf((*MockCharmHubClient)(nil).Refresh), arg0, arg1)
}
