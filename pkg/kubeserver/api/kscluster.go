package api

import "yunion.io/x/onecloud/pkg/apis"

type KsClusterCreateInput struct {
	apis.StatusDomainLevelResourceCreateInput

	IsSystem *bool  `json:"is_system"`
	Version  string `json:"version"`
	// imported cluster data
	ImportData *ImportClusterData `json:"import_data"`
}
