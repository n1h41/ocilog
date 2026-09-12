package oci

import (
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/identity"
	"github.com/oracle/oci-go-sdk/v65/logging"
	"github.com/oracle/oci-go-sdk/v65/loggingingestion"
	"github.com/oracle/oci-go-sdk/v65/loggingsearch"
)

// Client bundles the OCI identity and logging clients behind one type so the
// TUI only depends on this package rather than the SDK directly.
type Client struct {
	search   loggingsearch.LogSearchClient
	ingest   loggingingestion.LoggingClient
	manage   logging.LoggingManagementClient
	identity identity.IdentityClient
}

// NewClient builds a Client from the given configuration provider.
func NewClient(provider common.ConfigurationProvider) (*Client, error) {
	search, err := loggingsearch.NewLogSearchClientWithConfigurationProvider(provider)
	if err != nil {
		return nil, err
	}
	ingest, err := loggingingestion.NewLoggingClientWithConfigurationProvider(provider)
	if err != nil {
		return nil, err
	}
	manage, err := logging.NewLoggingManagementClientWithConfigurationProvider(provider)
	if err != nil {
		return nil, err
	}
	id, err := identity.NewIdentityClientWithConfigurationProvider(provider)
	if err != nil {
		return nil, err
	}
	return &Client{
		search:   search,
		ingest:   ingest,
		manage:   manage,
		identity: id,
	}, nil
}
