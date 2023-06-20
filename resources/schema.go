package resources

import (
	"context"
	"fmt"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/cloudquery/plugin-sdk/schema"
	"github.com/dihedron/cq-plugin-utils/format"
	"github.com/dihedron/cq-plugin-utils/pointer"
	"github.com/dihedron/cq-source-ldap/client"
	"github.com/dihedron/cq-source-ldap/resources/sid"
	"github.com/rs/zerolog"
)

// GetDynamicTables uses data in the spec section of the client configuration to
// dynamically build the information about the entity attributes being imported
// into columns.
func GetDynamicTables(ctx context.Context, meta schema.ClientMeta) (schema.Tables, error) {
	client := meta.(*client.Client)

	// get the table columns and populate the admission filter
	// for the main table
	tableColumns, err := buildTableColumnsSchema(client.Logger, &client.Specs.Table)
	if err != nil {
		client.Logger.Error().Err(err).Str("table", client.Specs.Table.Name).Msg("error getting table column schema and attributes")
		return nil, err
	}

	// now assemble the main table with its relations
	client.Logger.Debug().Msg("returning table schema")

	if client.Specs.Table.Description == nil {
		client.Specs.Table.Description = pointer.To(fmt.Sprintf("Table %q", client.Specs.Table.Name))
	}

	return []*schema.Table{
		{
			Name:        client.Specs.Table.Name,
			Description: *client.Specs.Table.Description,
			Resolver:    fetchTableData(&client.Specs.Table),
			Columns:     tableColumns,
		},
	}, nil
}

// buildTableColumnsSchema returns the schema definition of the given table's columns.
func buildTableColumnsSchema(logger zerolog.Logger, table *client.Table) ([]schema.Column, error) {
	columns := []schema.Column{}

	// prepare the template for value mapping
	funcMap := sprig.FuncMap()
	funcMap["toSID"] = func(data []byte) string {
		// sid conversion function
		return sid.New(data).String()
	}
	funcMap["toStrings"] = toStrings

	for _, c := range table.Columns {
		c := c
		logger.Debug().Str("table", table.Name).Str("name", c.Name).Msg("adding column...")

		mapping, err := template.New(c.Name).Funcs(funcMap).Parse(c.Mapping)
		if err != nil {
			logger.Error().Err(err).Str("table", table.Name).Str("column", c.Name).Str("mapping", c.Mapping).Msg("error parsing column mapping")
			return nil, fmt.Errorf("error parsing transform for column %q: %w", c.Name, err)
		}
		logger.Debug().Str("table", table.Name).Str("mapping definition", c.Mapping).Str("mapping kernel", format.ToJSON(mapping)).Msg("mapping parsed")

		if c.Description == nil {
			c.Description = pointer.To(fmt.Sprintf("The column mapping the %q field from the input data", c.Name))
		}
		column := schema.Column{
			Name:        c.Name,
			Description: *c.Description,
			Resolver:    fetchColumn(table, c.Name, mapping),
			CreationOptions: schema.ColumnCreationOptions{
				PrimaryKey: c.Key,
				Unique:     c.Unique,
				NotNull:    c.NotNull,
			},
		}
		// apply defaults if necessary
		if c.Type == nil {
			c.Type = pointer.To("string")
		}
		switch strings.ToLower(*c.Type) {
		case "string":
			logger.Debug().Str("table", table.Name).Str("name", c.Name).Msg("column is of type string")
			column.Type = schema.TypeString
		case "strings":
			logger.Debug().Str("table", table.Name).Str("name", c.Name).Msg("column is of type []string")
			column.Type = schema.TypeStringArray
		case "integer", "int":
			logger.Debug().Str("table", table.Name).Str("name", c.Name).Msg("column is of type int")
			column.Type = schema.TypeInt
		case "integers", "ints":
			logger.Debug().Str("table", table.Name).Str("name", c.Name).Msg("column is of type []ints")
			column.Type = schema.TypeIntArray
		case "boolean", "bool":
			logger.Debug().Str("table", table.Name).Str("name", c.Name).Msg("column is of type bool")
			column.Type = schema.TypeBool
		case "json":
			logger.Debug().Str("table", table.Name).Str("name", c.Name).Msg("column is of type json")
			column.Type = schema.TypeJSON
		default:
			logger.Debug().Str("table", table.Name).Str("name", c.Name).Msg("column is of unmapped type, assuming string")
			column.Type = schema.TypeString
		}
		columns = append(columns, column)
	}

	logger.Debug().Str("table", table.Name).Str("columns", format.ToJSON(columns)).Msg("returning columns schema and admission filter")
	return columns, nil
}
