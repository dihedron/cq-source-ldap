package client

type Spec struct {
	Endpoint string `json:"endpoint,omitempty" yaml:"endpoint,omitempty"`
	Username string `json:"username,omitempty" yaml:"username,omitempty"`
	Password string `json:"password,omitempty" yaml:"password,omitempty"`
	SkipTLS  bool   `json:"skiptls,omitempty" yaml:"skiptls,omitempty"`
	Query    Query  `json:"query,omitempty" yaml:"query,omitempty"`
	Table    Table  `json:"table,omitempty" yaml:"table,omitempty"`
}

type Query struct {
	BaseDN     string   `json:"basedn,omitempty" yaml:"basedn,omitempty"`
	Filter     string   `json:"filter,omitempty" yaml:"filter,omitempty"`
	Scope      *string  `json:"scope,omitempty" yaml:"scope,omitempty"`
	Attributes []string `json:"attributes,omitempty" yaml:"attributes,omitempty"`
}

type Column struct {
	Name        string  `json:"name,omitempty" yaml:"name,omitempty"`
	Description *string `json:"description,omitempty" yaml:"description,omitempty"`
	Type        *string `json:"type,omitempty" yaml:"type,omitempty"`
	Key         bool    `json:"key,omitempty" yaml:"pk,omitempty"`
	Unique      bool    `json:"unique,omitempty" yaml:"unique,omitempty"`
	NotNull     bool    `json:"notnull,omitempty" yaml:"notnull,omitempty"`
	Mapping     string  `json:"mapping,omitempty" yaml:"mapping,omitempty"`
}

type Table struct {
	Name        string    `json:"name,omitempty" yaml:"name,omitempty"`
	Description *string   `json:"description,omitempty" yaml:"description,omitempty"`
	Columns     []*Column `json:"columns,omitempty" yaml:"columns,omitempty"`
}
