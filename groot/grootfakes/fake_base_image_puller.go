// Code generated by counterfeiter. DO NOT EDIT.
package grootfakes

import (
	"sync"

	"code.cloudfoundry.org/grootfs/groot"
	"code.cloudfoundry.org/lager"
)

type FakeBaseImagePuller struct {
	PullStub        func(logger lager.Logger, spec groot.BaseImageSpec) (groot.BaseImage, error)
	pullMutex       sync.RWMutex
	pullArgsForCall []struct {
		logger lager.Logger
		spec   groot.BaseImageSpec
	}
	pullReturns struct {
		result1 groot.BaseImage
		result2 error
	}
	pullReturnsOnCall map[int]struct {
		result1 groot.BaseImage
		result2 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeBaseImagePuller) Pull(logger lager.Logger, spec groot.BaseImageSpec) (groot.BaseImage, error) {
	fake.pullMutex.Lock()
	ret, specificReturn := fake.pullReturnsOnCall[len(fake.pullArgsForCall)]
	fake.pullArgsForCall = append(fake.pullArgsForCall, struct {
		logger lager.Logger
		spec   groot.BaseImageSpec
	}{logger, spec})
	fake.recordInvocation("Pull", []interface{}{logger, spec})
	fake.pullMutex.Unlock()
	if fake.PullStub != nil {
		return fake.PullStub(logger, spec)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fake.pullReturns.result1, fake.pullReturns.result2
}

func (fake *FakeBaseImagePuller) PullCallCount() int {
	fake.pullMutex.RLock()
	defer fake.pullMutex.RUnlock()
	return len(fake.pullArgsForCall)
}

func (fake *FakeBaseImagePuller) PullArgsForCall(i int) (lager.Logger, groot.BaseImageSpec) {
	fake.pullMutex.RLock()
	defer fake.pullMutex.RUnlock()
	return fake.pullArgsForCall[i].logger, fake.pullArgsForCall[i].spec
}

func (fake *FakeBaseImagePuller) PullReturns(result1 groot.BaseImage, result2 error) {
	fake.PullStub = nil
	fake.pullReturns = struct {
		result1 groot.BaseImage
		result2 error
	}{result1, result2}
}

func (fake *FakeBaseImagePuller) PullReturnsOnCall(i int, result1 groot.BaseImage, result2 error) {
	fake.PullStub = nil
	if fake.pullReturnsOnCall == nil {
		fake.pullReturnsOnCall = make(map[int]struct {
			result1 groot.BaseImage
			result2 error
		})
	}
	fake.pullReturnsOnCall[i] = struct {
		result1 groot.BaseImage
		result2 error
	}{result1, result2}
}

func (fake *FakeBaseImagePuller) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.pullMutex.RLock()
	defer fake.pullMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeBaseImagePuller) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}

var _ groot.BaseImagePuller = new(FakeBaseImagePuller)
