package runner

import (
	"fmt"

	"code.cloudfoundry.org/grootfs/groot"
)

type InitSpec struct {
	Rootless         string
	UIDMappings      []groot.IDMappingSpec
	GIDMappings      []groot.IDMappingSpec
	StoreSizeBytes   int64
	FilesystemDriver string
}

func (r Runner) InitStore(spec InitSpec) error {
	args := []string{}

	for _, mapping := range spec.UIDMappings {
		args = append(args, "--uid-mapping",
			fmt.Sprintf("%d:%d:%d", mapping.NamespaceID, mapping.HostID, mapping.Size),
		)
	}

	for _, mapping := range spec.GIDMappings {
		args = append(args, "--gid-mapping",
			fmt.Sprintf("%d:%d:%d", mapping.NamespaceID, mapping.HostID, mapping.Size),
		)
	}

	if spec.Rootless != "" {
		args = append(args, "--rootless", spec.Rootless)
	}

	if spec.StoreSizeBytes > 0 {
		args = append(args, "--store-size-bytes", fmt.Sprintf("%d", spec.StoreSizeBytes))
	}

	if r.ExternaLogDeviceSize > 0 {
		args = append(args, "--external-logdev-size-mb", fmt.Sprintf("%d", r.ExternaLogDeviceSize))
	}

	if spec.FilesystemDriver != "" {
		args = append(args, "--driver", spec.FilesystemDriver)
	}

	_, err := r.RunSubcommand("init-store", args...)
	return err
}
