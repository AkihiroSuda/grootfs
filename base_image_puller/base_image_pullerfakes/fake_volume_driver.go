// Code generated by counterfeiter. DO NOT EDIT.
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
	volumePathReturnsOnCall map[int]struct {
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
	createVolumeReturnsOnCall map[int]struct {
		result1 string
		result2 error
	}
	FinalizeVolumeStub        func(logger lager.Logger, id string) error
	finalizeVolumeMutex       sync.RWMutex
	finalizeVolumeArgsForCall []struct {
		logger lager.Logger
		id     string
	}
	finalizeVolumeReturns struct {
		result1 error
	}
	finalizeVolumeReturnsOnCall map[int]struct {
		result1 error
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
	destroyVolumeReturnsOnCall map[int]struct {
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
	volumesReturnsOnCall map[int]struct {
		result1 []string
		result2 error
	}
	MoveVolumeStub        func(logger lager.Logger, from, to string) error
	moveVolumeMutex       sync.RWMutex
	moveVolumeArgsForCall []struct {
		logger lager.Logger
		from   string
		to     string
	}
	moveVolumeReturns struct {
		result1 error
	}
	moveVolumeReturnsOnCall map[int]struct {
		result1 error
	}
	WriteVolumeMetaStub        func(logger lager.Logger, id string, data base_image_puller.VolumeMeta) error
	writeVolumeMetaMutex       sync.RWMutex
	writeVolumeMetaArgsForCall []struct {
		logger lager.Logger
		id     string
		data   base_image_puller.VolumeMeta
	}
	writeVolumeMetaReturns struct {
		result1 error
	}
	writeVolumeMetaReturnsOnCall map[int]struct {
		result1 error
	}
	HandleOpaqueWhiteoutsStub        func(logger lager.Logger, id string, opaqueWhiteouts []string) error
	handleOpaqueWhiteoutsMutex       sync.RWMutex
	handleOpaqueWhiteoutsArgsForCall []struct {
		logger          lager.Logger
		id              string
		opaqueWhiteouts []string
	}
	handleOpaqueWhiteoutsReturns struct {
		result1 error
	}
	handleOpaqueWhiteoutsReturnsOnCall map[int]struct {
		result1 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeVolumeDriver) VolumePath(logger lager.Logger, id string) (string, error) {
	fake.volumePathMutex.Lock()
	ret, specificReturn := fake.volumePathReturnsOnCall[len(fake.volumePathArgsForCall)]
	fake.volumePathArgsForCall = append(fake.volumePathArgsForCall, struct {
		logger lager.Logger
		id     string
	}{logger, id})
	fake.recordInvocation("VolumePath", []interface{}{logger, id})
	fake.volumePathMutex.Unlock()
	if fake.VolumePathStub != nil {
		return fake.VolumePathStub(logger, id)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fake.volumePathReturns.result1, fake.volumePathReturns.result2
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

func (fake *FakeVolumeDriver) VolumePathReturnsOnCall(i int, result1 string, result2 error) {
	fake.VolumePathStub = nil
	if fake.volumePathReturnsOnCall == nil {
		fake.volumePathReturnsOnCall = make(map[int]struct {
			result1 string
			result2 error
		})
	}
	fake.volumePathReturnsOnCall[i] = struct {
		result1 string
		result2 error
	}{result1, result2}
}

func (fake *FakeVolumeDriver) CreateVolume(logger lager.Logger, parentID string, id string) (string, error) {
	fake.createVolumeMutex.Lock()
	ret, specificReturn := fake.createVolumeReturnsOnCall[len(fake.createVolumeArgsForCall)]
	fake.createVolumeArgsForCall = append(fake.createVolumeArgsForCall, struct {
		logger   lager.Logger
		parentID string
		id       string
	}{logger, parentID, id})
	fake.recordInvocation("CreateVolume", []interface{}{logger, parentID, id})
	fake.createVolumeMutex.Unlock()
	if fake.CreateVolumeStub != nil {
		return fake.CreateVolumeStub(logger, parentID, id)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fake.createVolumeReturns.result1, fake.createVolumeReturns.result2
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

func (fake *FakeVolumeDriver) CreateVolumeReturnsOnCall(i int, result1 string, result2 error) {
	fake.CreateVolumeStub = nil
	if fake.createVolumeReturnsOnCall == nil {
		fake.createVolumeReturnsOnCall = make(map[int]struct {
			result1 string
			result2 error
		})
	}
	fake.createVolumeReturnsOnCall[i] = struct {
		result1 string
		result2 error
	}{result1, result2}
}

func (fake *FakeVolumeDriver) FinalizeVolume(logger lager.Logger, id string) error {
	fake.finalizeVolumeMutex.Lock()
	ret, specificReturn := fake.finalizeVolumeReturnsOnCall[len(fake.finalizeVolumeArgsForCall)]
	fake.finalizeVolumeArgsForCall = append(fake.finalizeVolumeArgsForCall, struct {
		logger lager.Logger
		id     string
	}{logger, id})
	fake.recordInvocation("FinalizeVolume", []interface{}{logger, id})
	fake.finalizeVolumeMutex.Unlock()
	if fake.FinalizeVolumeStub != nil {
		return fake.FinalizeVolumeStub(logger, id)
	}
	if specificReturn {
		return ret.result1
	}
	return fake.finalizeVolumeReturns.result1
}

func (fake *FakeVolumeDriver) FinalizeVolumeCallCount() int {
	fake.finalizeVolumeMutex.RLock()
	defer fake.finalizeVolumeMutex.RUnlock()
	return len(fake.finalizeVolumeArgsForCall)
}

func (fake *FakeVolumeDriver) FinalizeVolumeArgsForCall(i int) (lager.Logger, string) {
	fake.finalizeVolumeMutex.RLock()
	defer fake.finalizeVolumeMutex.RUnlock()
	return fake.finalizeVolumeArgsForCall[i].logger, fake.finalizeVolumeArgsForCall[i].id
}

func (fake *FakeVolumeDriver) FinalizeVolumeReturns(result1 error) {
	fake.FinalizeVolumeStub = nil
	fake.finalizeVolumeReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeVolumeDriver) FinalizeVolumeReturnsOnCall(i int, result1 error) {
	fake.FinalizeVolumeStub = nil
	if fake.finalizeVolumeReturnsOnCall == nil {
		fake.finalizeVolumeReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.finalizeVolumeReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeVolumeDriver) DestroyVolume(logger lager.Logger, id string) error {
	fake.destroyVolumeMutex.Lock()
	ret, specificReturn := fake.destroyVolumeReturnsOnCall[len(fake.destroyVolumeArgsForCall)]
	fake.destroyVolumeArgsForCall = append(fake.destroyVolumeArgsForCall, struct {
		logger lager.Logger
		id     string
	}{logger, id})
	fake.recordInvocation("DestroyVolume", []interface{}{logger, id})
	fake.destroyVolumeMutex.Unlock()
	if fake.DestroyVolumeStub != nil {
		return fake.DestroyVolumeStub(logger, id)
	}
	if specificReturn {
		return ret.result1
	}
	return fake.destroyVolumeReturns.result1
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

func (fake *FakeVolumeDriver) DestroyVolumeReturnsOnCall(i int, result1 error) {
	fake.DestroyVolumeStub = nil
	if fake.destroyVolumeReturnsOnCall == nil {
		fake.destroyVolumeReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.destroyVolumeReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeVolumeDriver) Volumes(logger lager.Logger) ([]string, error) {
	fake.volumesMutex.Lock()
	ret, specificReturn := fake.volumesReturnsOnCall[len(fake.volumesArgsForCall)]
	fake.volumesArgsForCall = append(fake.volumesArgsForCall, struct {
		logger lager.Logger
	}{logger})
	fake.recordInvocation("Volumes", []interface{}{logger})
	fake.volumesMutex.Unlock()
	if fake.VolumesStub != nil {
		return fake.VolumesStub(logger)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fake.volumesReturns.result1, fake.volumesReturns.result2
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

func (fake *FakeVolumeDriver) VolumesReturnsOnCall(i int, result1 []string, result2 error) {
	fake.VolumesStub = nil
	if fake.volumesReturnsOnCall == nil {
		fake.volumesReturnsOnCall = make(map[int]struct {
			result1 []string
			result2 error
		})
	}
	fake.volumesReturnsOnCall[i] = struct {
		result1 []string
		result2 error
	}{result1, result2}
}

func (fake *FakeVolumeDriver) MoveVolume(logger lager.Logger, from string, to string) error {
	fake.moveVolumeMutex.Lock()
	ret, specificReturn := fake.moveVolumeReturnsOnCall[len(fake.moveVolumeArgsForCall)]
	fake.moveVolumeArgsForCall = append(fake.moveVolumeArgsForCall, struct {
		logger lager.Logger
		from   string
		to     string
	}{logger, from, to})
	fake.recordInvocation("MoveVolume", []interface{}{logger, from, to})
	fake.moveVolumeMutex.Unlock()
	if fake.MoveVolumeStub != nil {
		return fake.MoveVolumeStub(logger, from, to)
	}
	if specificReturn {
		return ret.result1
	}
	return fake.moveVolumeReturns.result1
}

func (fake *FakeVolumeDriver) MoveVolumeCallCount() int {
	fake.moveVolumeMutex.RLock()
	defer fake.moveVolumeMutex.RUnlock()
	return len(fake.moveVolumeArgsForCall)
}

func (fake *FakeVolumeDriver) MoveVolumeArgsForCall(i int) (lager.Logger, string, string) {
	fake.moveVolumeMutex.RLock()
	defer fake.moveVolumeMutex.RUnlock()
	return fake.moveVolumeArgsForCall[i].logger, fake.moveVolumeArgsForCall[i].from, fake.moveVolumeArgsForCall[i].to
}

func (fake *FakeVolumeDriver) MoveVolumeReturns(result1 error) {
	fake.MoveVolumeStub = nil
	fake.moveVolumeReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeVolumeDriver) MoveVolumeReturnsOnCall(i int, result1 error) {
	fake.MoveVolumeStub = nil
	if fake.moveVolumeReturnsOnCall == nil {
		fake.moveVolumeReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.moveVolumeReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeVolumeDriver) WriteVolumeMeta(logger lager.Logger, id string, data base_image_puller.VolumeMeta) error {
	fake.writeVolumeMetaMutex.Lock()
	ret, specificReturn := fake.writeVolumeMetaReturnsOnCall[len(fake.writeVolumeMetaArgsForCall)]
	fake.writeVolumeMetaArgsForCall = append(fake.writeVolumeMetaArgsForCall, struct {
		logger lager.Logger
		id     string
		data   base_image_puller.VolumeMeta
	}{logger, id, data})
	fake.recordInvocation("WriteVolumeMeta", []interface{}{logger, id, data})
	fake.writeVolumeMetaMutex.Unlock()
	if fake.WriteVolumeMetaStub != nil {
		return fake.WriteVolumeMetaStub(logger, id, data)
	}
	if specificReturn {
		return ret.result1
	}
	return fake.writeVolumeMetaReturns.result1
}

func (fake *FakeVolumeDriver) WriteVolumeMetaCallCount() int {
	fake.writeVolumeMetaMutex.RLock()
	defer fake.writeVolumeMetaMutex.RUnlock()
	return len(fake.writeVolumeMetaArgsForCall)
}

func (fake *FakeVolumeDriver) WriteVolumeMetaArgsForCall(i int) (lager.Logger, string, base_image_puller.VolumeMeta) {
	fake.writeVolumeMetaMutex.RLock()
	defer fake.writeVolumeMetaMutex.RUnlock()
	return fake.writeVolumeMetaArgsForCall[i].logger, fake.writeVolumeMetaArgsForCall[i].id, fake.writeVolumeMetaArgsForCall[i].data
}

func (fake *FakeVolumeDriver) WriteVolumeMetaReturns(result1 error) {
	fake.WriteVolumeMetaStub = nil
	fake.writeVolumeMetaReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeVolumeDriver) WriteVolumeMetaReturnsOnCall(i int, result1 error) {
	fake.WriteVolumeMetaStub = nil
	if fake.writeVolumeMetaReturnsOnCall == nil {
		fake.writeVolumeMetaReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.writeVolumeMetaReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeVolumeDriver) HandleOpaqueWhiteouts(logger lager.Logger, id string, opaqueWhiteouts []string) error {
	var opaqueWhiteoutsCopy []string
	if opaqueWhiteouts != nil {
		opaqueWhiteoutsCopy = make([]string, len(opaqueWhiteouts))
		copy(opaqueWhiteoutsCopy, opaqueWhiteouts)
	}
	fake.handleOpaqueWhiteoutsMutex.Lock()
	ret, specificReturn := fake.handleOpaqueWhiteoutsReturnsOnCall[len(fake.handleOpaqueWhiteoutsArgsForCall)]
	fake.handleOpaqueWhiteoutsArgsForCall = append(fake.handleOpaqueWhiteoutsArgsForCall, struct {
		logger          lager.Logger
		id              string
		opaqueWhiteouts []string
	}{logger, id, opaqueWhiteoutsCopy})
	fake.recordInvocation("HandleOpaqueWhiteouts", []interface{}{logger, id, opaqueWhiteoutsCopy})
	fake.handleOpaqueWhiteoutsMutex.Unlock()
	if fake.HandleOpaqueWhiteoutsStub != nil {
		return fake.HandleOpaqueWhiteoutsStub(logger, id, opaqueWhiteouts)
	}
	if specificReturn {
		return ret.result1
	}
	return fake.handleOpaqueWhiteoutsReturns.result1
}

func (fake *FakeVolumeDriver) HandleOpaqueWhiteoutsCallCount() int {
	fake.handleOpaqueWhiteoutsMutex.RLock()
	defer fake.handleOpaqueWhiteoutsMutex.RUnlock()
	return len(fake.handleOpaqueWhiteoutsArgsForCall)
}

func (fake *FakeVolumeDriver) HandleOpaqueWhiteoutsArgsForCall(i int) (lager.Logger, string, []string) {
	fake.handleOpaqueWhiteoutsMutex.RLock()
	defer fake.handleOpaqueWhiteoutsMutex.RUnlock()
	return fake.handleOpaqueWhiteoutsArgsForCall[i].logger, fake.handleOpaqueWhiteoutsArgsForCall[i].id, fake.handleOpaqueWhiteoutsArgsForCall[i].opaqueWhiteouts
}

func (fake *FakeVolumeDriver) HandleOpaqueWhiteoutsReturns(result1 error) {
	fake.HandleOpaqueWhiteoutsStub = nil
	fake.handleOpaqueWhiteoutsReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeVolumeDriver) HandleOpaqueWhiteoutsReturnsOnCall(i int, result1 error) {
	fake.HandleOpaqueWhiteoutsStub = nil
	if fake.handleOpaqueWhiteoutsReturnsOnCall == nil {
		fake.handleOpaqueWhiteoutsReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.handleOpaqueWhiteoutsReturnsOnCall[i] = struct {
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
	fake.finalizeVolumeMutex.RLock()
	defer fake.finalizeVolumeMutex.RUnlock()
	fake.destroyVolumeMutex.RLock()
	defer fake.destroyVolumeMutex.RUnlock()
	fake.volumesMutex.RLock()
	defer fake.volumesMutex.RUnlock()
	fake.moveVolumeMutex.RLock()
	defer fake.moveVolumeMutex.RUnlock()
	fake.writeVolumeMetaMutex.RLock()
	defer fake.writeVolumeMetaMutex.RUnlock()
	fake.handleOpaqueWhiteoutsMutex.RLock()
	defer fake.handleOpaqueWhiteoutsMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
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
