package main

import (
	"errors"
	"github.com/yourusername/open-z3950-gateway/pkg/provider"
	"github.com/yourusername/open-z3950-gateway/pkg/z3950"
)

type MockProvider struct {
	SearchFunc                 func(db string, query z3950.StructuredQuery) ([]string, error)
	FetchFunc                  func(db string, ids []string) ([]*z3950.MARCRecord, error)
	ScanFunc                   func(db, field, startTerm string) ([]provider.ScanResult, error)
	CreateILLRequestFunc       func(req provider.ILLRequest) error
	GetILLRequestFunc          func(id int64) (*provider.ILLRequest, error)
	ListILLRequestsFunc        func() ([]provider.ILLRequest, error)
	UpdateILLRequestStatusFunc func(id int64, status string) error
	CreateUserFunc             func(user *provider.User) error
	GetUserByUsernameFunc      func(username string) (*provider.User, error)
	CreateTargetFunc           func(target *provider.Target) error
	ListTargetsFunc            func() ([]provider.Target, error)
	DeleteTargetFunc           func(id int64) error
	GetTargetByNameFunc        func(name string) (*provider.Target, error)
}

func (m *MockProvider) Search(db string, query z3950.StructuredQuery) ([]string, error) {
	if m.SearchFunc != nil {
		return m.SearchFunc(db, query)
	}
	return nil, nil
}

func (m *MockProvider) Fetch(db string, ids []string) ([]*z3950.MARCRecord, error) {
	if m.FetchFunc != nil {
		return m.FetchFunc(db, ids)
	}
	return nil, nil
}

func (m *MockProvider) Scan(db, field, startTerm string) ([]provider.ScanResult, error) {
	if m.ScanFunc != nil {
		return m.ScanFunc(db, field, startTerm)
	}
	return nil, nil
}

func (m *MockProvider) CreateILLRequest(req provider.ILLRequest) error {
	if m.CreateILLRequestFunc != nil {
		return m.CreateILLRequestFunc(req)
	}
	return nil
}

func (m *MockProvider) GetILLRequest(id int64) (*provider.ILLRequest, error) {
	if m.GetILLRequestFunc != nil {
		return m.GetILLRequestFunc(id)
	}
	return nil, nil
}

func (m *MockProvider) ListILLRequests() ([]provider.ILLRequest, error) {
	if m.ListILLRequestsFunc != nil {
		return m.ListILLRequestsFunc()
	}
	return nil, nil
}

func (m *MockProvider) UpdateILLRequestStatus(id int64, status string) error {
	if m.UpdateILLRequestStatusFunc != nil {
		return m.UpdateILLRequestStatusFunc(id, status)
	}
	return nil
}

func (m *MockProvider) CreateUser(user *provider.User) error {
	if m.CreateUserFunc != nil {
		return m.CreateUserFunc(user)
	}
	return nil
}

func (m *MockProvider) GetUserByUsername(username string) (*provider.User, error) {
	if m.GetUserByUsernameFunc != nil {
		return m.GetUserByUsernameFunc(username)
	}
	return nil, errors.New("user not found")
}

func (m *MockProvider) CreateTarget(target *provider.Target) error {
	if m.CreateTargetFunc != nil {
		return m.CreateTargetFunc(target)
	}
	return nil
}

func (m *MockProvider) ListTargets() ([]provider.Target, error) {
	if m.ListTargetsFunc != nil {
		return m.ListTargetsFunc()
	}
	return nil, nil
}

func (m *MockProvider) DeleteTarget(id int64) error {
	if m.DeleteTargetFunc != nil {
		return m.DeleteTargetFunc(id)
	}
	return nil
}

func (m *MockProvider) GetTargetByName(name string) (*provider.Target, error) {
	if m.GetTargetByNameFunc != nil {
		return m.GetTargetByNameFunc(name)
	}
	return nil, errors.New("target not found")
}