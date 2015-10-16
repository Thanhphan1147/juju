// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package api

import (
	"github.com/juju/juju/workload"
	"gopkg.in/juju/charm.v5"
)

// FullInfo2api converts a workload.FullInfo struct into an api.FullPayload struct.
func Payload2api(p workload.Payload) Payload {
	tags := make([]string, len(p.Tags))
	copy(tags, p.Tags)
	return Payload{
		Class:   p.Name,
		Type:    p.Type,
		ID:      p.ID,
		Status:  p.Status,
		Tags:    tags,
		Unit:    p.Unit,
		Machine: p.Machine,
	}
}

// API2FullInfo converts an API Payload info struct into a workload.FullInfo struct.
func API2Payload(apiInfo Payload) workload.Payload {
	tags := make([]string, len(apiInfo.Tags))
	copy(tags, apiInfo.Tags)
	return workload.Payload{
		PayloadClass: charm.PayloadClass{
			Name: apiInfo.Class,
			Type: apiInfo.Type,
		},
		ID:      apiInfo.ID,
		Status:  apiInfo.Status,
		Tags:    tags,
		Unit:    apiInfo.Unit,
		Machine: apiInfo.Machine,
	}
}
