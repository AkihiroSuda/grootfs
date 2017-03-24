package runner

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"syscall"

	"code.cloudfoundry.org/grootfs/groot"
	"code.cloudfoundry.org/grootfs/store/image_cloner"
)

func (r Runner) Create(spec groot.CreateSpec) (groot.Image, error) {
	if r.skippingMount() {
		if err := r.skipMount(); err != nil {
			return groot.Image{}, err
		}
		r.Json = true
	}

	output, err := r.create(spec)
	if err != nil {
		return groot.Image{}, err
	}

	rawOutput := output

	if r.skippingMount() {
		var imageInfo image_cloner.ImageInfo
		if err := json.Unmarshal([]byte(output), &imageInfo); err != nil {
			return groot.Image{}, err
		}

		if err := syscall.Mount(imageInfo.Mount.Source, imageInfo.Mount.Destination, imageInfo.Mount.Type, 0, imageInfo.Mount.Options[0]); err != nil {
			return groot.Image{}, err
		}

		output = filepath.Dir(imageInfo.Rootfs)
	}

	return groot.Image{
		Path:       output,
		RootFSPath: filepath.Join(output, "rootfs"),
		Json:       rawOutput,
	}, nil
}

func (r Runner) create(spec groot.CreateSpec) (string, error) {
	args := r.makeCreateArgs(spec)
	output, err := r.RunSubcommand("create", args...)
	if err != nil {
		return "", err
	}

	return output, nil
}

func (r Runner) skippingMount() bool {
	return r.Driver == "overlay-xfs" && r.SysCredential.Uid != 0
}

func (r Runner) makeCreateArgs(spec groot.CreateSpec) []string {
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

	if r.CleanOnCreate || r.NoCleanOnCreate {
		if r.CleanOnCreate {
			args = append(args, "--with-clean")
		}
		if r.NoCleanOnCreate {
			args = append(args, "--without-clean")
		}
	} else {
		if spec.CleanOnCreate {
			args = append(args, "--with-clean")
		} else {
			args = append(args, "--without-clean")
		}
	}

	if r.Json || r.NoJson {
		if r.Json {
			args = append(args, "--json")
		}
		if r.NoJson {
			args = append(args, "--no-json")
		}
	} else {
		if spec.Json {
			args = append(args, "--json")
		}
	}

	if r.InsecureRegistry != "" {
		args = append(args, "--insecure-registry", r.InsecureRegistry)
	}

	if r.RegistryUsername != "" {
		args = append(args, "--username", r.RegistryUsername)
	}

	if r.RegistryPassword != "" {
		args = append(args, "--password", r.RegistryPassword)
	}

	if spec.DiskLimit != 0 {
		args = append(args, "--disk-limit-size-bytes",
			strconv.FormatInt(spec.DiskLimit, 10),
		)
		if spec.ExcludeBaseImageFromQuota {
			args = append(args, "--exclude-image-from-quota")
		}
	}

	if spec.BaseImage != "" {
		args = append(args, spec.BaseImage)
	}

	if spec.ID != "" {
		args = append(args, spec.ID)
	}

	return args
}
