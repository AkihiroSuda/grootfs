package idmapping_parser

import (
	"fmt"

	"code.cloudfoundry.org/grootfs/groot"
)

type flagIdMappingParser struct {
	mappings []string
}

func NewFlagMappingParser(mappings []string) IdMappingParser {
	return &flagIdMappingParser{
		mappings: mappings,
	}
}

func (f *flagIdMappingParser) ParseMappings() ([]groot.IDMappingSpec, error) {
	mappings := []groot.IDMappingSpec{}

	for _, v := range f.mappings {
		var mapping groot.IDMappingSpec
		_, err := fmt.Sscanf(v, "%d:%d:%d", &mapping.NamespaceID, &mapping.HostID, &mapping.Size)
		if err != nil {
			return nil, err
		}
		mappings = append(mappings, mapping)
	}

	return mappings, nil
}
