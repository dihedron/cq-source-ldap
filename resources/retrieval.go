package resources

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"text/template"

	"github.com/antonmedv/expr"
	"github.com/antonmedv/expr/vm"
	"github.com/cloudquery/plugin-sdk/schema"
	"github.com/dihedron/cq-plugin-utils/format"
	"github.com/dihedron/cq-source-ldap/client"
	"github.com/go-ldap/ldap/v3"
)

const PagingSize = 100

// fetchTableData reads the main table's data by reading it from the input file and
// unmarshallilng it into a set of rows using format-specific mechanisms, then
// encodes the information as a map[string]any per row and returns it; fetchColumn
// knows how to pick the data out of this map and set it into the resource being
// returned to ClouqQuery.
func fetchTableData(table *client.MainTable, evaluator *vm.Program) func(ctx context.Context, meta schema.ClientMeta, parent *schema.Resource, res chan<- interface{}) error {

	return func(ctx context.Context, meta schema.ClientMeta, parent *schema.Resource, res chan<- interface{}) error {
		client := meta.(*client.Client)
		client.Logger.Debug().Str("table", table.Name).Msg("fetching data...")

		baseDN := client.Specs.Query.BaseDN // "DC=example,DC=com"
		filter := client.Specs.Query.Filter // "(CN=A1234567)"
		scope := ldap.ScopeWholeSubtree
		if client.Specs.Query.Scope != nil {
			switch strings.TrimSpace(strings.ToLower(*client.Specs.Query.Scope)) {
			case "base":
				scope = ldap.ScopeBaseObject
			default:
				scope = ldap.ScopeWholeSubtree
			}
		}

		// prepare the set of attributes (columns) to retrieve
		attributes := []string{}
		for _, c := range table.Columns {
			if c.Attribute != nil {
				attributes = append(attributes, c.Attribute.Name)
			} else {
				attributes = append(attributes, c.Name)
			}
		}
		request := ldap.NewSearchRequest(baseDN, scope, 0, 0, 0, false, filter, attributes, []ldap.Control{})

		results, err := client.Client.SearchWithPaging(request, PagingSize)
		if err != nil {
			client.Logger.Error().Err(err).Msg("error querying LDAP server")
			return fmt.Errorf("failed to query LDAP: %w", err)
		}

		client.Logger.Debug().Int("entries", len(results.Entries)).Msg("query complete")

		for _, result := range results.Entries {
			result := result

			// collect all the entry attributes into a map
			attributes := map[string][]string{
				"dn": {
					result.DN,
				},
			}
			for _, attribute := range result.Attributes {
				attributes[attribute.Name] = attribute.Values
			}

			accepted := true
			if evaluator != nil {
				client.Logger.Debug().Msg("evaluator is not nil")
				accepted = false
				env := Env{
					"_": attributes,
				}

				if output, err := expr.Run(evaluator, env); err != nil {
					client.Logger.Error().Err(err).Msg("error running evaluator")
				} else {
					client.Logger.Debug().Any("output", output).Msg("received output")
					accepted = output.(bool)
				}
			} else {
				client.Logger.Warn().Msg("evaluator is nil")
			}

			if accepted {
				//client.Logger.Debug().Str("filter", *table.Filter).Str("entry", format.ToJSON(entry)).Msg("accepting entry")
				client.Logger.Debug().Str("attributes", format.ToJSON(attributes)).Msg("accepting entry")
				res <- attributes
			} else {
				//client.Logger.Debug().Str("filter", *table.Filter).Str("entry", format.ToJSON(entry)).Msg("rejecting entry")
				client.Logger.Debug().Str("attributes", format.ToJSON(attributes)).Msg("rejecting entry")
			}
		}

		return nil
	}
}

func fetchRelationData(table *client.Table, admitter *vm.Program) func(ctx context.Context, meta schema.ClientMeta, parent *schema.Resource, res chan<- interface{}) error {

	return func(ctx context.Context, meta schema.ClientMeta, parent *schema.Resource, res chan<- interface{}) error {
		client := meta.(*client.Client)

		// grab the parent row and use it to extract the
		// columns that go into the child relation
		entry := parent.Item.(map[string]any)

		client.Logger.Debug().Str("table", table.Name).Str("entry", format.ToJSON(entry)).Msg("fetching data from parent...")

		accepted := true
		if admitter != nil {
			accepted = false
			env := Env{
				"_": entry,
			}

			if output, err := expr.Run(admitter, env); err != nil {
				client.Logger.Error().Err(err).Msg("error running evaluator")
			} else {
				client.Logger.Debug().Any("output", output).Msg("received output")
				accepted = output.(bool)
			}
		}

		if accepted {
			if admitter != nil {
				client.Logger.Debug().Str("filter", *table.Filter).Str("row", format.ToJSON(entry)).Msg("accepting entry")
			} else {
				client.Logger.Debug().Str("row", format.ToJSON(entry)).Msg("passing on row")
			}
			res <- entry
		} else {
			client.Logger.Debug().Str("filter", *table.Filter).Str("row", format.ToJSON(entry)).Msg("rejecting entry")
		}

		return nil
	}
}

// fetchColumn picks the value under the right key from the map[string]any
// and sets it into the resource being returned to CloudQuery.
func fetchColumn(table *client.Table, name string, transform *template.Template, attributeName string, attributeType AttributeType) func(ctx context.Context, meta schema.ClientMeta, resource *schema.Resource, c schema.Column) error {

	return func(ctx context.Context, meta schema.ClientMeta, resource *schema.Resource, c schema.Column) error {
		client := meta.(*client.Client)
		attributes := resource.Item.(map[string][]string)

		client.Logger.Debug().Str("table", table.Name).Str("column", c.Name).Str("attribute", attributeName).Str("entry", format.ToJSON(attributes)).Msg("retrieving column for table")

		var value any
	loop:
		for name := range attributes {
			if strings.EqualFold(name, attributeName) {
				values := attributes[name]
				switch c.Type {
				case schema.TypeString:
					switch len(values) {
					case 0:
						value = nil
					case 1:
						value = values[0]
					default:
						value = fmt.Sprintf("%v", attributes[name])
					}
				case schema.TypeStringArray:
					value = values
				default:
					client.Logger.Error().Int("type", int(c.Type)).Msg("unsupported field type")
				}
				break loop
			}
		}

		client.Logger.Debug().Str("value", fmt.Sprintf("%v", value)).Str("type", fmt.Sprintf("%T", value)).Msg("checking value type")

		// now apply the transform if it is available
		for _, spec := range table.Columns {
			if strings.EqualFold(spec.Name, c.Name) && transform != nil {
				client.Logger.Debug().Msg("applying transform...")
				var buffer bytes.Buffer
				target := struct {
					Name       string
					Value      any
					Type       schema.ValueType
					Attributes map[string][]string
				}{
					Name:       c.Name,
					Value:      value,
					Type:       c.Type,
					Attributes: attributes,
				}
				if err := transform.Execute(&buffer, target); err != nil {
					client.Logger.Error().Err(err).Any("value", value).Str("transform", *spec.Transform).Any("attributes", attributes).Msg("error applying transform")
					return err
				}
				value = buffer.String()
				break
			}
		}

		client.Logger.Debug().Any("value", value).Msg("after transform...")

		// if value == nil {
		// 	client.Logger.Warn().Msg("value is nil")
		// 	if c.CreationOptions.NotNull {
		// 		err := fmt.Errorf("invalid nil value for non-nullable column %s", c.Name)
		// 		client.Logger.Error().Err(err).Str("name", c.Name).Msg("error setting column")
		// 		return err
		// 	}
		// } else {
		// 	client.Logger.Warn().Msg("value is NOT nil")
		// 	if reflect.ValueOf(value).IsZero() {
		// 		if !c.CreationOptions.NotNull {
		// 			// column is nullable, let's null it
		// 			client.Logger.Warn().Str("name", c.Name).Msg("nulling column value")
		// 			value = nil
		// 		} else {
		// 			client.Logger.Warn().Msg("set default value for type")
		// 			switch c.Type {
		// 			case schema.TypeBool:
		// 				value = false
		// 			case schema.TypeInt:
		// 				value = 0
		// 			case schema.TypeString:
		// 				value = ""
		// 			}
		// 		}
		// 	}
		// }
		// in XLSX some values may be null, in which case we must
		// be sure we're not asking cloudQuery to parse invalid values
		return resource.Set(c.Name, value)
	}
}
