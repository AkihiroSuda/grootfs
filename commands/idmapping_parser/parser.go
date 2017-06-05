package idmapping_parser

import "code.cloudfoundry.org/grootfs/groot"

type IdMappingParser interface {
	ParseMappings() ([]groot.IDMappingSpec, error)
}
