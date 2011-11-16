package juju

import "launchpad.net/juju/go/schema"

// A EnvironProvider represents a computing and storage provider.
type EnvironProvider interface {
	// ConfigChecker is used to check sections of the environments.yaml
	// file that specify this provider. The value passed to the Checker is
	// that returned from the yaml parse, of type schema.MapType.
	ConfigChecker() schema.Checker

	// NewEnviron creates a new Environ with
	// the given attributes returned by the ConfigChecker.
	// The name is that given in environments.yaml.
	Open(name string, attributes interface{}) (Environ, error)
}

// Instance represents a running machine instance.
type Instance interface {
	// Id returns a provider-generated identifier for the Instance.
	// N.B. this does not return the machine identifier.
	Id() string
	DNSName() string
}

// An Environ represents a juju environment as specified
// in the environments.yaml file.
type Environ interface {
	// Bootstrap initializes the new environment.
	Bootstrap() error

	// StartInstance asks for a new machine instance to be created,
	// associated with the provided machine identifier
	// TODO add arguments to specify type of new machine
	// and zookeeper instances.
	StartInstance(machineId string) (Instance, error)

	// StopInstance shuts down the given Instance.
	StopInstances([]Instance) error

	// Instances returns the list of currently started instances.
	Instances() ([]Instance, error)

	// Destroy shuts down all known machines and destroys the
	// rest of the environment.
	Destroy() error
}
