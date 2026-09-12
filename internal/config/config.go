package config

import (
	"os"

	"github.com/oracle/oci-go-sdk/v65/common"
)

// Config holds the OCI settings needed to talk to the logging services.
type Config struct {
	// Region overrides the region from the default OCI config when set.
	Region string
	// CompartmentOCID narrows list/search operations to a specific compartment.
	// When empty the tenancy OCID is used.
	CompartmentOCID string
}

// Load builds a Config from the environment.
func Load() (Config, error) {
	return Config{
		Region:          os.Getenv("OCI_REGION"),
		CompartmentOCID: os.Getenv("OCI_COMPARTMENT_OCID"),
	}, nil
}

// regionProvider wraps a configuration provider, overriding only its region.
type regionProvider struct {
	common.ConfigurationProvider
	region string
}

func (p regionProvider) Region() (string, error) { return p.region, nil }

// Provider returns the default OCI configuration provider, honouring the
// region override in cfg.
func (c Config) Provider() (common.ConfigurationProvider, error) {
	provider := common.DefaultConfigProvider()
	if c.Region != "" {
		provider = regionProvider{ConfigurationProvider: provider, region: c.Region}
	}
	return provider, nil
}
