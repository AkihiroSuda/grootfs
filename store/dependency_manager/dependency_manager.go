package dependency_manager

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"
)

type DependencyManager struct {
	storePath string
}

func NewDependencyManager(storePath string) *DependencyManager {
	return &DependencyManager{
		storePath: storePath,
	}
}

func (d *DependencyManager) Register(id string, chainIDs []string) error {
	if err := os.MkdirAll(filepath.Join(d.storePath, "images", id, "refs"), 0755); err != nil {
		return err
	}

	for _, vol := range chainIDs {
		f, err := os.Create(filepath.Join(d.storePath, "meta", fmt.Sprintf("%s-ref-counter", vol)))
		if err != nil {
			return err
		}
		f.Close()

		err = os.Link(filepath.Join(d.storePath, "meta", fmt.Sprintf("%s-ref-counter", vol)),
			filepath.Join(d.storePath, "images", id, "refs", fmt.Sprintf("%s", vol)))
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *DependencyManager) Deregister(id string) error {
	return os.Remove(filepath.Join(d.storePath, "images", id, "refs"))
}

func (d *DependencyManager) Dependencies(id string) ([]string, error) {
	infos, err := ioutil.ReadDir(filepath.Join(d.storePath, "images", id, "refs"))
	if err != nil {
		return nil, err
	}

	deps := []string{}
	for _, info := range infos {
		deps = append(deps, info.Name())
	}

	return deps, nil
}

func (d *DependencyManager) Referenced(id string) (bool, error) {
	refCount, err := numReferences(filepath.Join(d.storePath, "meta", fmt.Sprintf("%s-ref-counter", id)))
	if err != nil {
		return false, err
	}
	return refCount > 1, nil
}

func numReferences(filename string) (uint64, error) {
	nlink := uint64(0)

	fileInfo, err := os.Stat(filename)
	if err != nil {
		return nlink, err
	}
	if sys := fileInfo.Sys(); sys != nil {
		if stat, ok := sys.(*syscall.Stat_t); ok {
			nlink = stat.Nlink
		}
	}
	return nlink, nil
}
