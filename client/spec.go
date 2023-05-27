package client

import (
	"text/template"

	"github.com/antonmedv/expr/vm"
)

type Column struct {
	Name        string             `json:"name,omitempty" yaml:"name,omitempty"`
	Description *string            `json:"description,omitempty" yaml:"description,omitempty"`
	Type        string             `json:"type,omitempty" yaml:"type,omitempty"`
	Key         bool               `json:"key,omitempty" yaml:"pk,omitempty"`
	Unique      bool               `json:"unique,omitempty" yaml:"unique,omitempty"`
	NotNull     bool               `json:"notnull,omitempty" yaml:"notnull,omitempty"`
	Transform   *string            `json:"transform,omitempty" yaml:"transform,omitempty"`
	Template    *template.Template `json:"-" yaml:"-"`
}

type Table struct {
	Name        string      `json:"name,omitempty" yaml:"name,omitempty"`
	Description *string     `json:"description,omitempty" yaml:"description,omitempty"`
	Filter      *string     `json:"filter,omitempty" yaml:"filter,omitempty"`
	Evaluator   *vm.Program `json:"-,omitempty" yaml:"-,omitempty"`
	Columns     []*Column   `json:"columns,omitempty" yaml:"columns,omitempty"`
}
type Spec struct {
	Endpoint     string  `json:"endpoint,omitempty" yaml:"endpoint,omitempty"`
	Username     string  `json:"username,omitempty" yaml:"username,omitempty"`
	Password     string  `json:"password,omitempty" yaml:"password,omitempty"`
	SearchFilter string  `json:"searchfilter,omitempty" yaml:"searchfilter,omitempty"`
	SearchScope  *string `json:"searchscope,omitempty" yaml:"searchscope,omitempty"`
	Table        Table   `json:"table,omitempty" yaml:"table,omitempty"`
	Relations    []Table `json:"relations,omitempty" yaml:"relations,omitempty"`
}
