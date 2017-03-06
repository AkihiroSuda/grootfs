// This file was generated by counterfeiter
package base_image_pullerfakes

import (
	"sync"

	"code.cloudfoundry.org/grootfs/base_image_puller"
	"code.cloudfoundry.org/lager"
)

type FakeVolumeDriver struct {
	VolumePathStub        func(logger lager.Logger, id string) (string, error)
	volumePathMutex       sync.RWMutex
	volumePathArgsForCall []struct {
		logger lager.Logger
		id     string
	}
	volumePathReturns struct {
		result1 string
		result2 error
	}
	CreateVolumeStub        func(logger lager.Logger, parentID, id string) (string, error)
	createVolumeMutex       sync.RWMutex
	createVolumeArgsForCall []struct {
		logger   lager.Logger
		parentID string
		id       string
	}
	createVolumeReturns struct {
		result1 string
		result2 error
	}
	DestroyVolumeStub        func(logger lager.Logger, id string) error
	destroyVolumeMutex       sync.RWMutex
	destroyVolumeArgsForCall []struct {
		logger lager.Logger
		id     string
	}
	destroyVolumeReturns struct {
		result1 error
	}
	VolumesStub        func(logger lager.Logger) ([]string, error)
	volumesMutex       sync.RWMutex
	volumesArgsForCall []struct {
		logger lager.Logger
	}
	volumesReturns struct {
		result1 []string
		result2 error
	}
	MoveVolumeStub        func(from, to string) error
	moveVolumeMutex       sync.RWMutex
	moveVolumeArgsForCall []struct {
		from string
		to   string
	}
	moveVolumeReturns struct {
		result1 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeVolumeDriver) VolumePath(logger lager.Logger, id string) (string, error) {
	fake.volumePathMutex.Lock()
	fake.volumePathArgsForCall = append(fake.volumePathArgsForCall, struct {
		logger lager.Logger
		id     string
	}{logger, id})
	fake.recordInvocation("VolumePath", []interface{}{logger, id})
	fake.volumePathMutex.Unlock()
	if fake.VolumePathStub != nil {
		return fake.VolumePathStub(logger, id)
	} else {
		return fake.volumePathReturns.result1, fake.volumePathReturns.result2
	}
}

func (fake *FakeVolumeDriver) VolumePathCallCount() int {
	fake.volumePathMutex.RLock()
	defer fake.volumePathMutex.RUnlock()
	return len(fake.volumePathArgsForCall)
}

func (fake *FakeVolumeDriver) VolumePathArgsForCall(i int) (lager.Logger, string) {
	fake.volumePathMutex.RLock()
	defer fake.volumePathMutex.RUnlock()
	return fake.volumePathArgsForCall[i].logger, fake.volumePathArgsForCall[i].id
}

func (fake *FakeVolumeDriver) VolumePathReturns(result1 string, result2 error) {
	fake.VolumePathStub = nil
	fake.volumePathReturns = struct {
		result1 string
		result2 error
	}{result1, result2}
}

func (fake *FakeVolumeDriver) CreateVolume(logger lager.Logger, parentID string, id string) (string, error) {
	fake.createVolumeMutex.Lock()
	fake.createVolumeArgsForCall = append(fake.createVolumeArgsForCall, struct {
		logger   lager.Logger
		parentID string
		id       string
	}{logger, parentID, id})
	fake.recordInvocation("CreateVolume", []interface{}{logger, parentID, id})
	fake.createVolumeMutex.Unlock()
	if fake.CreateVolumeStub != nil {
		return fake.CreateVolumeStub(logger, parentID, id)
	} else {
		return fake.createVolumeReturns.result1, fake.createVolumeReturns.result2
	}
}

func (fake *FakeVolumeDriver) CreateVolumeCallCount() int {
	fake.createVolumeMutex.RLock()
	defer fake.createVolumeMutex.RUnlock()
	return len(fake.createVolumeArgsForCall)
}

func (fake *FakeVolumeDriver) CreateVolumeArgsForCall(i int) (lager.Logger, string, string) {
	fake.createVolumeMutex.RLock()
	defer fake.createVolumeMutex.RUnlock()
	return fake.createVolumeArgsForCall[i].logger, fake.createVolumeArgsForCall[i].parentID, fake.createVolumeArgsForCall[i].id
}

func (fake *FakeVolumeDriver) CreateVolumeReturns(result1 string, result2 error) {
	fake.CreateVolumeStub = nil
	fake.createVolumeReturns = struct {
		result1 string
		result2 error
	}{result1, result2}
}

func (fake *FakeVolumeDriver) DestroyVolume(logger lager.Logger, id string) error {
	fake.destroyVolumeMutex.Lock()
	fake.destroyVolumeArgsForCall = append(fake.destroyVolumeArgsForCall, struct {
		logger lager.Logger
		id     string
	}{logger, id})
	fake.recordInvocation("DestroyVolume", []interface{}{logger, id})
	fake.destroyVolumeMutex.Unlock()
	if fake.DestroyVolumeStub != nil {
		return fake.DestroyVolumeStub(logger, id)
	} else {
		return fake.destroyVolumeReturns.result1
	}
}

func (fake *FakeVolumeDriver) DestroyVolumeCallCount() int {
	fake.destroyVolumeMutex.RLock()
	defer fake.destroyVolumeMutex.RUnlock()
	return len(fake.destroyVolumeArgsForCall)
}

func (fake *FakeVolumeDriver) DestroyVolumeArgsForCall(i int) (lager.Logger, string) {
	fake.destroyVolumeMutex.RLock()
	defer fake.destroyVolumeMutex.RUnlock()
	return fake.destroyVolumeArgsForCall[i].logger, fake.destroyVolumeArgsForCall[i].id
}

func (fake *FakeVolumeDriver) DestroyVolumeReturns(result1 error) {
	fake.DestroyVolumeStub = nil
	fake.destroyVolumeReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeVolumeDriver) Volumes(logger lager.Logger) ([]string, error) {
	fake.volumesMutex.Lock()
	fake.volumesArgsForCall = append(fake.volumesArgsForCall, struct {
		logger lager.Logger
	}{logger})
	fake.recordInvocation("Volumes", []interface{}{logger})
	fake.volumesMutex.Unlock()
	if fake.VolumesStub != nil {
		return fake.VolumesStub(logger)
	} else {
		return fake.volumesReturns.result1, fake.volumesReturns.result2
	}
}

func (fake *FakeVolumeDriver) VolumesCallCount() int {
	fake.volumesMutex.RLock()
	defer fake.volumesMutex.RUnlock()
	return len(fake.volumesArgsForCall)
}

func (fake *FakeVolumeDriver) VolumesArgsForCall(i int) lager.Logger {
	fake.volumesMutex.RLock()
	defer fake.volumesMutex.RUnlock()
	return fake.volumesArgsForCall[i].logger
}

func (fake *FakeVolumeDriver) VolumesReturns(result1 []string, result2 error) {
	fake.VolumesStub = nil
	fake.volumesReturns = struct {
		result1 []string
		result2 error
	}{result1, result2}
}

func (fake *FakeVolumeDriver) MoveVolume(from string, to string) error {
	fake.moveVolumeMutex.Lock()
	fake.moveVolumeArgsForCall = append(fake.moveVolumeArgsForCall, struct {
		from string
		to   string
	}{from, to})
	fake.recordInvocation("MoveVolume", []interface{}{from, to})
	fake.moveVolumeMutex.Unlock()
	if fake.MoveVolumeStub != nil {
		return fake.MoveVolumeStub(from, to)
	} else {
		return fake.moveVolumeReturns.result1
	}
}

func (fake *FakeVolumeDriver) MoveVolumeCallCount() int {
	fake.moveVolumeMutex.RLock()
	defer fake.moveVolumeMutex.RUnlock()
	return len(fake.moveVolumeArgsForCall)
}

func (fake *FakeVolumeDriver) MoveVolumeArgsForCall(i int) (string, string) {
	fake.moveVolumeMutex.RLock()
	defer fake.moveVolumeMutex.RUnlock()
	return fake.moveVolumeArgsForCall[i].from, fake.moveVolumeArgsForCall[i].to
}

func (fake *FakeVolumeDriver) MoveVolumeReturns(result1 error) {
	fake.MoveVolumeStub = nil
	fake.moveVolumeReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeVolumeDriver) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.volumePathMutex.RLock()
	defer fake.volumePathMutex.RUnlock()
	fake.createVolumeMutex.RLock()
	defer fake.createVolumeMutex.RUnlock()
	fake.destroyVolumeMutex.RLock()
	defer fake.destroyVolumeMutex.RUnlock()
	fake.volumesMutex.RLock()
	defer fake.volumesMutex.RUnlock()
	fake.moveVolumeMutex.RLock()
	defer fake.moveVolumeMutex.RUnlock()
	return fake.invocations
}

func (fake *FakeVolumeDriver) recordInvocation(key string, args []interface{}) {
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

var _ base_image_puller.VolumeDriver = new(FakeVolumeDriver)
