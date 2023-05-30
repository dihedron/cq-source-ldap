package client

import (
	"context"
	"crypto/tls"
	"fmt"

	"github.com/cloudquery/plugin-sdk/plugins/source"
	"github.com/cloudquery/plugin-sdk/schema"
	"github.com/cloudquery/plugin-sdk/specs"
	"github.com/dihedron/cq-plugin-utils/format"
	"github.com/go-ldap/ldap/v3"
	"github.com/rs/zerolog"
)

type Client struct {
	Logger zerolog.Logger
	Specs  *Spec
	Client *ldap.Conn
}

func (c *Client) ID() string {
	return "github.com/dihedron/cq-source-ldap"
}

func New(ctx context.Context, logger zerolog.Logger, s specs.Source, opts source.Options) (schema.ClientMeta, error) {
	var pluginSpec Spec

	if err := s.UnmarshalSpec(&pluginSpec); err != nil {
		return nil, fmt.Errorf("failed to unmarshal plugin spec: %w", err)
	}

	logger.Debug().Str("spec", format.ToJSON(pluginSpec)).Msg("plugin spec")

	logger.Debug().Str("endpoint", pluginSpec.Endpoint).Msg("connecting to LDAP server...")
	var dialOpts []ldap.DialOpt
	if pluginSpec.SkipTLS {
		dialOpts = []ldap.DialOpt{
			ldap.DialWithTLSConfig(
				&tls.Config{
					InsecureSkipVerify: true,
				},
			),
		}
	}
	client, err := ldap.DialURL(pluginSpec.Endpoint, dialOpts...)
	if err != nil {
		logger.Error().Err(err).Str("endpoint", pluginSpec.Endpoint).Msg("error connecting to LDAP server")
		return nil, err
	}
	logger.Info().Str("endpoint", pluginSpec.Endpoint).Msg("connected to LDAP server")

	logger.Debug().Str("username", pluginSpec.Username).Msg("binding to LDAP server...")
	err = client.Bind(pluginSpec.Username, pluginSpec.Password)
	if err != nil {
		logger.Error().Err(err).Str("endpoint", pluginSpec.Endpoint).Str("username", pluginSpec.Username).Msg("error binding to LDAP server")
		defer client.Close()
		return nil, err
	}

	logger.Info().Str("endpoint", pluginSpec.Endpoint).Str("username", pluginSpec.Username).Msg("bound to LDAP server")
	return &Client{
		Logger: logger,
		Specs:  &pluginSpec,
		Client: client,
	}, nil
}
