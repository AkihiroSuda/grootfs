package runner

import (
	"fmt"

	"code.cloudfoundry.org/grootfs/store/manager"
)

func (r Runner) InitStore(spec manager.InitSpec) error {
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

	if spec.StoreSizeBytes > 0 {
		args = append(args, "--store-size-bytes", fmt.Sprintf("%d", spec.StoreSizeBytes))
	}

	if r.ExternaLogDeviceSize > 0 {
		args = append(args, "--external-logdev-size-mb", fmt.Sprintf("%d", r.ExternaLogDeviceSize))
	}

	if r.RootlessUser != "" && r.RootlessGroup != "" {
		args = append(args, "--rootless", fmt.Sprintf("%s:%s", r.RootlessUser, r.RootlessGroup))
	}

	_, err := r.RunSubcommand("init-store", args...)
	return err
}
