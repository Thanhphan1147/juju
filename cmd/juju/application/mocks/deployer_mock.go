// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/juju/juju/cmd/juju/application/deployer (interfaces: Deployer,DeployerFactory)
//
// Generated by this command:
//
//	mockgen -package mocks -destination mocks/deployer_mock.go github.com/juju/juju/cmd/juju/application/deployer Deployer,DeployerFactory
//

// Package mocks is a generated GoMock package.
package mocks

import (
	reflect "reflect"

	cmd "github.com/juju/cmd/v3"
	deployer "github.com/juju/juju/cmd/juju/application/deployer"
	gomock "go.uber.org/mock/gomock"
)

// MockDeployer is a mock of Deployer interface.
type MockDeployer struct {
	ctrl     *gomock.Controller
	recorder *MockDeployerMockRecorder
}

// MockDeployerMockRecorder is the mock recorder for MockDeployer.
type MockDeployerMockRecorder struct {
	mock *MockDeployer
}

// NewMockDeployer creates a new mock instance.
func NewMockDeployer(ctrl *gomock.Controller) *MockDeployer {
	mock := &MockDeployer{ctrl: ctrl}
	mock.recorder = &MockDeployerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockDeployer) EXPECT() *MockDeployerMockRecorder {
	return m.recorder
}

// PrepareAndDeploy mocks base method.
func (m *MockDeployer) PrepareAndDeploy(arg0 *cmd.Context, arg1 deployer.DeployerAPI, arg2 deployer.Resolver) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PrepareAndDeploy", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// PrepareAndDeploy indicates an expected call of PrepareAndDeploy.
func (mr *MockDeployerMockRecorder) PrepareAndDeploy(arg0, arg1, arg2 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PrepareAndDeploy", reflect.TypeOf((*MockDeployer)(nil).PrepareAndDeploy), arg0, arg1, arg2)
}

// String mocks base method.
func (m *MockDeployer) String() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "String")
	ret0, _ := ret[0].(string)
	return ret0
}

// String indicates an expected call of String.
func (mr *MockDeployerMockRecorder) String() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "String", reflect.TypeOf((*MockDeployer)(nil).String))
}

// MockDeployerFactory is a mock of DeployerFactory interface.
type MockDeployerFactory struct {
	ctrl     *gomock.Controller
	recorder *MockDeployerFactoryMockRecorder
}

// MockDeployerFactoryMockRecorder is the mock recorder for MockDeployerFactory.
type MockDeployerFactoryMockRecorder struct {
	mock *MockDeployerFactory
}

// NewMockDeployerFactory creates a new mock instance.
func NewMockDeployerFactory(ctrl *gomock.Controller) *MockDeployerFactory {
	mock := &MockDeployerFactory{ctrl: ctrl}
	mock.recorder = &MockDeployerFactoryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockDeployerFactory) EXPECT() *MockDeployerFactoryMockRecorder {
	return m.recorder
}

// GetDeployer mocks base method.
func (m *MockDeployerFactory) GetDeployer(arg0 deployer.DeployerConfig, arg1 deployer.ModelConfigGetter, arg2 deployer.Resolver) (deployer.Deployer, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetDeployer", arg0, arg1, arg2)
	ret0, _ := ret[0].(deployer.Deployer)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetDeployer indicates an expected call of GetDeployer.
func (mr *MockDeployerFactoryMockRecorder) GetDeployer(arg0, arg1, arg2 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetDeployer", reflect.TypeOf((*MockDeployerFactory)(nil).GetDeployer), arg0, arg1, arg2)
}
