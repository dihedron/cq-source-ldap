package resources

import (
	"context"
	"fmt"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/antonmedv/expr"
	"github.com/antonmedv/expr/vm"
	"github.com/cloudquery/plugin-sdk/schema"
	"github.com/dihedron/cq-plugin-utils/format"
	"github.com/dihedron/cq-plugin-utils/pointer"
	"github.com/dihedron/cq-source-ldap/client"
	"github.com/rs/zerolog"
)

// GetDynamicTables uses data in the spec section of the client configuration to
// dynamically build the information about the entity attributes being imported
// into columns.
func GetDynamicTables(ctx context.Context, meta schema.ClientMeta) (schema.Tables, error) {
	client := meta.(*client.Client)

	// get the table columns and populate the admission filter
	// for the main table
	tableColumns, tableFilter, err := buildTableColumnsSchema(client.Logger, &client.Specs.Table.Table)
	if err != nil {
		client.Logger.Error().Err(err).Str("table", client.Specs.Table.Name).Msg("error getting table column schema and attributes")
		return nil, err
	}

	// now loop over and add relations
	relations := []*schema.Table{}
	client.Logger.Debug().Str("table", client.Specs.Table.Name).Msg("adding relations...")
	for _, relation := range client.Specs.Table.Relations {

		relation := relation

		relationColumns, relationFilter, err := buildTableColumnsSchema(client.Logger, &relation)
		if err != nil {
			client.Logger.Error().Err(err).Str("table", relation.Name).Msg("error getting relation column schema")
			return nil, err
		}

		client.Logger.Debug().Str("relation", relation.Name).Msg("adding relation to schema")

		if relation.Description == nil {
			relation.Description = pointer.To(fmt.Sprintf("Table %q", relation.Name))
		}

		relations = append(relations, &schema.Table{
			Name:        relation.Name,
			Description: *relation.Description,
			Resolver:    fetchRelationData(&relation, relationFilter),
			Columns:     relationColumns,
		})
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
			Resolver:    fetchTableData(&client.Specs.Table, tableFilter),
			Columns:     tableColumns,
			Relations:   relations,
		},
	}, nil
}

// buildTableColumnsSchema returns the schema definition of the given table's columns
// and populates the table's Evaluator field if the Filter is not null (side effect).
// TODO: fix side effect once working
func buildTableColumnsSchema(logger zerolog.Logger, table *client.Table) ([]schema.Column, *vm.Program, error) {
	var err error

	entry := map[string]any{}
	// start by looping over the columns definitions and creating the Column schema
	// object; while looping over the columns, we are also creating a map holding
	// the column names and an example (zero) value for each column, which we'll use
	// when initialising the admission filter, which expects to work on a given data
	// structure when being compiled; moreover, we build a map that allows us to
	// know which attribute to take to populate each column (since column name and
	// LDAP attribute names may not be the same, e.g. givenName may be mapped onto
	// Name and fn onto Surname), and how to extract data from it (e.g. some fields
	// like Active Directory SIDs are binary encodings that need specific parsing
	// to be rendered as human-readable strings).
	columns := []schema.Column{}

	for _, c := range table.Columns {
		c := c
		logger.Debug().Str("table", table.Name).Str("name", c.Name).Msg("adding column...")

		// prepare the template for value transformation if there is a transform
		var transform *template.Template

		if c.Transform != nil {
			transform, err = template.New(c.Name).Funcs(sprig.FuncMap()).Parse(*c.Transform)
			if err != nil {
				logger.Error().Err(err).Str("table", table.Name).Str("column", c.Name).Str("transform", *c.Transform).Msg("error parsing column transform")
				return nil, nil, fmt.Errorf("error parsing transform for column %q: %w", c.Name, err)
			}
			logger.Debug().Str("table", table.Name).Str("template", format.ToJSON(transform)).Str("transform", *c.Transform).Msg("template after having parsed transform")
		}

		if c.Description == nil {
			c.Description = pointer.To(fmt.Sprintf("The column mapping the %q field from the input data", c.Name))
		}
		column := schema.Column{
			Name:        c.Name,
			Description: *c.Description,
			Resolver:    fetchColumn(table, c.Name, transform, getAttributeName(c), getAttributeType(c), c.Split),
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
		if c.Attribute == nil {
			c.Attribute = &client.Attribute{
				Name: c.Name,
				Type: pointer.To("string"),
			}
		}
		switch strings.ToLower(*c.Type) {
		case "string":
			logger.Debug().Str("table", table.Name).Str("name", c.Name).Msg("column is of type string")
			column.Type = schema.TypeString
			entry[c.Name] = ""
		case "[]string":
			logger.Debug().Str("table", table.Name).Str("name", c.Name).Msg("column is of type []string")
			column.Type = schema.TypeStringArray
			entry[c.Name] = []string{}
		case "integer", "int":
			logger.Debug().Str("table", table.Name).Str("name", c.Name).Msg("column is of type int")
			column.Type = schema.TypeInt
			entry[c.Name] = 0
		case "boolean", "bool":
			logger.Debug().Str("table", table.Name).Str("name", c.Name).Msg("column is of type bool")
			column.Type = schema.TypeBool
			entry[c.Name] = false
		case "sid":
			logger.Debug().Str("table", table.Name).Str("name", c.Name).Msg("column is of type sid")
			column.Type = schema.TypeString
			entry[c.Name] = ""
		default:
			logger.Debug().Str("table", table.Name).Str("name", c.Name).Msg("column is of unmapped type, assuming string")
			column.Type = schema.TypeString
			entry[c.Name] = ""
		}
		columns = append(columns, column)
	}

	// now initialise the filter using the row map that we've populated above;
	var filter *vm.Program
	if table.Filter != nil {
		logger.Debug().Str("table", table.Name).Str("filter", *table.Filter).Str("entry template", format.ToJSON(entry)).Msg("compiling row filter")
		env := Env{
			"_": entry,
			"string": func(v any) string {
				return fmt.Sprintf("%v", v)
			},
		}
		if filter, err = expr.Compile(*table.Filter, expr.Env(env), expr.AsBool(), expr.Operator(".", "GetAttribute")); err != nil {
			filter = nil // just make sure
			logger.Error().Err(err).Str("table", table.Name).Str("filter", *table.Filter).Msg("error compiling expression evaluator")
		} else {
			logger.Debug().Str("table", table.Name).Str("filter", *table.Filter).Msg("expression evaluator successfully compiled")
		}
	}

	logger.Debug().Str("table", table.Name).Str("columns", format.ToJSON(columns)).Msg("returning columns schema and admission filter")
	return columns, filter, nil
}

type Env map[string]any

func (Env) GetAttribute(entry map[string]any, attribute string) any {
	for name := range entry {
		if strings.EqualFold(name, attribute) {
			return entry[name]
		}
	}
	return nil
}
