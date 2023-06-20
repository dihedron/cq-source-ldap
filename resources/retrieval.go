package resources

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"text/template"

	"github.com/cloudquery/plugin-sdk/v3/schema"
	"github.com/dihedron/cq-plugin-utils/format"
	"github.com/dihedron/cq-source-ldap/client"
	"github.com/dop251/goja"
	"github.com/go-ldap/ldap/v3"
)

const PagingSize = 100

// fetchTableData reads the main table's data by performing an LDAP query;
// the attributes to be retrieved from the query are specified in the Spec.
// Actual formatting of output values is performed by the fetchColumn function.
func fetchTableData(table *client.Table) func(ctx context.Context, meta schema.ClientMeta, parent *schema.Resource, channel chan<- interface{}) error {

	return func(ctx context.Context, meta schema.ClientMeta, parent *schema.Resource, channel chan<- interface{}) error {
		client := meta.(*client.Client)
		client.Logger.Debug().Str("table", table.Name).Msg("fetching data...")

		baseDN := client.Specs.Query.BaseDN // "DC=example,DC=com"
		query := client.Specs.Query.Query   // "(CN=A1234567)"
		scope := ldap.ScopeWholeSubtree
		if client.Specs.Query.Scope != nil {
			switch strings.TrimSpace(strings.ToLower(*client.Specs.Query.Scope)) {
			case "base":
				scope = ldap.ScopeBaseObject
			default:
				scope = ldap.ScopeWholeSubtree
			}
		}

		request := ldap.NewSearchRequest(baseDN, scope, 0, 0, 0, false, query, client.Specs.Query.Attributes, []ldap.Control{})

		results, err := client.Client.SearchWithPaging(request, PagingSize)
		if err != nil {
			client.Logger.Error().Err(err).Msg("error querying LDAP server")
			return fmt.Errorf("failed to query LDAP: %w", err)
		}

		client.Logger.Debug().Int("entries", len(results.Entries)).Msg("query complete")

		var vm *goja.Runtime
		var accept goja.Callable
		if client.Specs.Query.Filter != nil {
			vm = goja.New()
			vm.Set("toString", toString)
			vm.Set("toStrings", toStrings)
			vm.Set("log", makeLog(client.Logger))
			// TODO: add further helper functions here
			if _, err := vm.RunString(*client.Specs.Query.Filter); err != nil {
				client.Logger.Error().Err(err).Str("filter", *client.Specs.Query.Filter).Msg("error parsing filter")
				return err
			}
			var ok bool
			if accept, ok = goja.AssertFunction(vm.Get("accept")); !ok {
				client.Logger.Error().Err(err).Str("filter", *client.Specs.Query.Filter).Msg("no valid definition of 'accept' in filter")
				return err
			}
		}

		for _, result := range results.Entries {
			result := result

			// collect all the entry attributes into a map
			attributes := map[string][][]byte{
				"dn": {
					[]byte(result.DN),
				},
			}
			for _, attribute := range result.Attributes {
				attributes[strings.ToLower(attribute.Name)] = attribute.ByteValues
			}

			accepted := true
			if accept != nil {
				client.Logger.Info().Bytes("cn", attributes["cn"][0]).Msg("running filter against entry")
				res, err := accept(goja.Undefined(), vm.ToValue(attributes))
				if err != nil {
					client.Logger.Error().Err(err).Msg("error running accept on entry")
					continue
				}
				if b := res.ToBoolean(); b {
					client.Logger.Info().Any("result", res).Msg("filter accepted entry")
					accepted = true
				} else {
					client.Logger.Info().Any("result", res).Msg("filter rejected entry")
					accepted = false
				}
			}

			if accepted {
				channel <- attributes
			}
		}

		return nil
	}
}

// fetchColumn picks the value for the given column by applying the
// provided mapping; the Golang template must be correct and safe, no
// check or business logic is applied here. Since the template has
// access to the whole entity, the column value can be a combination
// of attributes.
func fetchColumn(table *client.Table, name string, mapping *template.Template) func(ctx context.Context, meta schema.ClientMeta, resource *schema.Resource, c schema.Column) error {

	return func(ctx context.Context, meta schema.ClientMeta, resource *schema.Resource, c schema.Column) error {
		client := meta.(*client.Client)
		attributes := resource.Item.(map[string][][]byte)

		client.Logger.Debug().Str("table", table.Name).Str("column", c.Name).Str("mapping", format.ToJSON(mapping)).Str("attributes", format.ToJSON(attributes)).Msg("retrieving column by applying mapping...")

		var buffer bytes.Buffer
		if err := mapping.Execute(&buffer, attributes); err != nil {
			client.Logger.Error().Err(err).Str("table", table.Name).Str("column", c.Name).Str("mapping", format.ToJSON(mapping)).Any("attributes", attributes).Msg("error applying mapping")
			return err
		}
		client.Logger.Debug().Str("table", table.Name).Str("column", c.Name).Str("value", buffer.String()).Msg("after mapping...")

		return resource.Set(c.Name, buffer.Bytes())
	}
}
