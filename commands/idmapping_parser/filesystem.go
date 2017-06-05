package idmapping_parser

import "code.cloudfoundry.org/grootfs/groot"

type filesystemIdMappingParser struct {
}

func (f *filesystemIdMappingParser) ParseMappings() ([]groot.IDMappingSpec, error) {
	return nil, nil
}
