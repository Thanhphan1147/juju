// Copyright 2020 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package common

import (
	"github.com/juju/errors"
	"github.com/juju/loggo"

	"github.com/juju/juju/charmhub"
	"github.com/juju/juju/environs/config"
)

// ModelGetter defines an interface for getting a model.
type ModelGetter interface {
	Model() (ConfigModel, error)
}

// ConfigModel defines an interface for getting the config of a model.
type ConfigModel interface {
	Config() (*config.Config, error)
}

// CharmhubClient creates a new charmhub Client based on this model's config.
func CharmhubClient(mg ModelGetter, httpClient charmhub.HTTPClient, logger loggo.Logger) (*charmhub.Client, error) {
	model, err := mg.Model()
	if err != nil {
		return nil, errors.Trace(err)
	}
	modelConfig, err := model.Config()
	if err != nil {
		return nil, errors.Trace(err)
	}
	url, _ := modelConfig.CharmHubURL()

	cfg := charmhub.Config{
		URL:        url,
		HTTPClient: httpClient,
		Logger:     logger,
	}

	client, err := charmhub.NewClient(cfg)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return client, nil
}
