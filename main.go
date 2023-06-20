package main

import (
	"github.com/cloudquery/plugin-sdk/v3/serve"
	"github.com/dihedron/cq-source-ldap/plugin"
)

func main() {
	serve.Source(plugin.Plugin())
}
