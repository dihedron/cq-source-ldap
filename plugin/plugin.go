package plugin

import (
	"github.com/cloudquery/plugin-sdk/plugins/source"
	"github.com/dihedron/cq-source-ldap/client"
	"github.com/dihedron/cq-source-ldap/resources"
)

var (
	Version = "development"
)

func Plugin() *source.Plugin {
	return source.NewPlugin(
		"github.com/dihedron-ldap",
		Version,
		nil, // no static tables
		client.New,
		source.WithDynamicTableOption(resources.GetDynamicTables),
	)
}
