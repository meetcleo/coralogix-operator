// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/coralogix/coralogix-operator/controllers/clientset (interfaces: DashboardsClientInterface)
//
// Generated by this command:
//
//	mockgen -destination=../mock_clientset/mock_dashboards-client.go -package=mock_clientset github.com/coralogix/coralogix-operator/controllers/clientset DashboardsClientInterface
//

// Package mock_clientset is a generated GoMock package.
package mock_clientset

import (
	context "context"
	reflect "reflect"

	v1 "github.com/coralogix/coralogix-operator/controllers/clientset/grpc/coralogix-dashboards/v1"
	gomock "go.uber.org/mock/gomock"
)

// MockDashboardsClientInterface is a mock of DashboardsClientInterface interface.
type MockDashboardsClientInterface struct {
	ctrl     *gomock.Controller
	recorder *MockDashboardsClientInterfaceMockRecorder
}

// MockDashboardsClientInterfaceMockRecorder is the mock recorder for MockDashboardsClientInterface.
type MockDashboardsClientInterfaceMockRecorder struct {
	mock *MockDashboardsClientInterface
}

// NewMockDashboardsClientInterface creates a new mock instance.
func NewMockDashboardsClientInterface(ctrl *gomock.Controller) *MockDashboardsClientInterface {
	mock := &MockDashboardsClientInterface{ctrl: ctrl}
	mock.recorder = &MockDashboardsClientInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockDashboardsClientInterface) EXPECT() *MockDashboardsClientInterfaceMockRecorder {
	return m.recorder
}

// CreateDashboard mocks base method.
func (m *MockDashboardsClientInterface) CreateDashboard(arg0 context.Context, arg1 *v1.CreateDashboardRequest) (*v1.CreateDashboardResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateDashboard", arg0, arg1)
	ret0, _ := ret[0].(*v1.CreateDashboardResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateDashboard indicates an expected call of CreateDashboard.
func (mr *MockDashboardsClientInterfaceMockRecorder) CreateDashboard(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateDashboard", reflect.TypeOf((*MockDashboardsClientInterface)(nil).CreateDashboard), arg0, arg1)
}

// DeleteDashboard mocks base method.
func (m *MockDashboardsClientInterface) DeleteDashboard(arg0 context.Context, arg1 *v1.DeleteDashboardRequest) (*v1.DeleteDashboardResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteDashboard", arg0, arg1)
	ret0, _ := ret[0].(*v1.DeleteDashboardResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DeleteDashboard indicates an expected call of DeleteDashboard.
func (mr *MockDashboardsClientInterfaceMockRecorder) DeleteDashboard(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteDashboard", reflect.TypeOf((*MockDashboardsClientInterface)(nil).DeleteDashboard), arg0, arg1)
}

// GetDashboard mocks base method.
func (m *MockDashboardsClientInterface) GetDashboard(arg0 context.Context, arg1 *v1.GetDashboardRequest) (*v1.GetDashboardResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetDashboard", arg0, arg1)
	ret0, _ := ret[0].(*v1.GetDashboardResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetDashboard indicates an expected call of GetDashboard.
func (mr *MockDashboardsClientInterfaceMockRecorder) GetDashboard(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetDashboard", reflect.TypeOf((*MockDashboardsClientInterface)(nil).GetDashboard), arg0, arg1)
}

// UpdateDashboard mocks base method.
func (m *MockDashboardsClientInterface) UpdateDashboard(arg0 context.Context, arg1 *v1.ReplaceDashboardRequest) (*v1.ReplaceDashboardResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateDashboard", arg0, arg1)
	ret0, _ := ret[0].(*v1.ReplaceDashboardResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateDashboard indicates an expected call of UpdateDashboard.
func (mr *MockDashboardsClientInterfaceMockRecorder) UpdateDashboard(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateDashboard", reflect.TypeOf((*MockDashboardsClientInterface)(nil).UpdateDashboard), arg0, arg1)
}
