// Copyright 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package common

import (
	"launchpad.net/juju-core/state/api/params"
)

// AddressAndCertGetter can be used to find out
// state server addresses and the CA public certificate.
type AddressAndCertGetter interface {
	Addresses() ([]string, error)
	APIAddressesFromMachines() ([]string, error)
	CACert() []byte
	APIHostPorts() ([][]instance.HostPort, error)
	WatchAPIHostPorts() state.NotifyWatcher
}

// APIAddresser implements the APIAddresses method
type APIAddresser struct {
	resources         *Resources
	getter AddressAndCertGetter
}

// NewAPIAddresser returns a new APIAddresser that uses the given getter to
// fetch its addresses.
func NewAPIAddresser(getter AddressAndCertGetter, resources *Resources) *APIAddresser {
	return &APIAddresser{
		getter: getter,
		resources: resources,
	}
}

// APIHostPorts returns the API server addresses.
func (api *APIAddresser) APIHostPorts() (params.APIHostPortsResults, error) {
	servers, err := api.getter.APIHostPorts()
	if err != nil {
		return params.APIHostPortsResults{}, nil
	}
	return APIHostPortsResults{
		Servers: servers,
	}, nil
}

// WatchAPIHostPorts watches the API server addresses.
func (api *APIAddresser) WatchAPIHostPorts() (params.NotifyWatchResult, error) {
	watch := api.getter.WatchAPIHostPorts()
	if _, ok := <-watch.Changes(); ok {
		return params.NotifyWatchResult{
			NotifyWatcherId: a.resources.Register(watch),
		}, nil
	}
	return params.NotifyWatchResult{}, watcher.MustErr(watch)
}

// APIAddresses returns the list of addresses used to connect to the API.
func (a *APIAddresser) APIAddresses() (params.StringsResult, error) {
	// TODO(rog) change this to use api.st.APIHostPorts()
	addrs, err := a.getter.APIAddressesFromMachines()
	if err != nil {
		return params.StringsResult{}, err
	}
	return params.StringsResult{
		Result: addrs,
	}, nil
}

// StateAddresser implements a common set of methods for getting state
// server addresses, and the CA certificate used to authenticate them.
type StateAddresser struct {
	getter AddressAndCertGetter
}

// NewAddresser returns a new StateAddresser that uses the given
// st value to fetch its addresses.
func NewStateAddresser(getter AddressAndCertGetter) *StateAddresser {
	return &StateAddresser{getter}
}

// StateAddresses returns the list of addresses used to connect to the state.
func (a *StateAddresser) StateAddresses() (params.StringsResult, error) {
	addrs, err := a.getter.Addresses()
	if err != nil {
		return params.StringsResult{}, err
	}
	return params.StringsResult{
		Result: addrs,
	}, nil
}

// CACert returns the certificate used to validate the state connection.
// Note: there is an open bug that Uniter (which uses only APIAddresser) should
// add CACert to its interface. When it does, this API si likely to move to
// APIAddresser instead of StateAddresser. (All other users of StateAddresser
// already also expose APIAddresser)
func (a *StateAddresser) CACert() params.BytesResult {
	return params.BytesResult{
		Result: a.getter.CACert(),
	}
}
