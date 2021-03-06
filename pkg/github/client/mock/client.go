// Copyright 2022 The Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

// Code generated by MockGen. DO NOT EDIT.
// Source: octorun.github.io/octorun/pkg/github (interfaces: Client)

// Package mock_github is a generated GoMock package.
package mock_github

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	client "octorun.github.io/octorun/pkg/github/client"
)

// MockClient is a mock of Client interface.
type MockClient struct {
	ctrl     *gomock.Controller
	recorder *MockClientMockRecorder
}

// MockClientMockRecorder is the mock recorder for MockClient.
type MockClientMockRecorder struct {
	mock *MockClient
}

// NewMockClient creates a new mock instance.
func NewMockClient(ctrl *gomock.Controller) *MockClient {
	mock := &MockClient{ctrl: ctrl}
	mock.recorder = &MockClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockClient) EXPECT() *MockClientMockRecorder {
	return m.recorder
}

// CreateRunnerToken mocks base method.
func (m *MockClient) CreateRunnerToken(arg0 context.Context, arg1 string) (client.RunnerToken, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateRunnerToken", arg0, arg1)
	ret0, _ := ret[0].(client.RunnerToken)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateRunnerToken indicates an expected call of CreateRunnerToken.
func (mr *MockClientMockRecorder) CreateRunnerToken(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateRunnerToken", reflect.TypeOf((*MockClient)(nil).CreateRunnerToken), arg0, arg1)
}

// GetRunner mocks base method.
func (m *MockClient) GetRunner(arg0 context.Context, arg1 string, arg2 int64) (client.Runner, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRunner", arg0, arg1, arg2)
	ret0, _ := ret[0].(client.Runner)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetRunner indicates an expected call of GetRunner.
func (mr *MockClientMockRecorder) GetRunner(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRunner", reflect.TypeOf((*MockClient)(nil).GetRunner), arg0, arg1, arg2)
}
