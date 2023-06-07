package client

type Attribute struct {
	Name string  `json:"name,omitempty" yaml:"name,omitempty"`
	Type *string `json:"type,omitempty" yaml:"type,omitempty"`
}

type Column struct {
	Name        string     `json:"name,omitempty" yaml:"name,omitempty"`
	Description *string    `json:"description,omitempty" yaml:"description,omitempty"`
	Type        *string    `json:"type,omitempty" yaml:"type,omitempty"`
	Attribute   *Attribute `json:"attribute,omitempty" yaml:"attribute,omitempty"` // is absent, we use the name and string
	Key         bool       `json:"key,omitempty" yaml:"pk,omitempty"`
	Unique      bool       `json:"unique,omitempty" yaml:"unique,omitempty"`
	NotNull     bool       `json:"notnull,omitempty" yaml:"notnull,omitempty"`
	Transform   *string    `json:"transform,omitempty" yaml:"transform,omitempty"`
	Split       *bool      `json:"split,omitempty" yaml:"split,omitempty"` // only relations
}

type Table struct {
	Name        string    `json:"name,omitempty" yaml:"name,omitempty"`
	Description *string   `json:"description,omitempty" yaml:"description,omitempty"`
	Filter      *string   `json:"filter,omitempty" yaml:"filter,omitempty"`
	Columns     []*Column `json:"columns,omitempty" yaml:"columns,omitempty"`
}

type MainTable struct {
	Table
	Relations []Table `json:"relations,omitempty" yaml:"relations,omitempty"`
}

type Spec struct {
	Endpoint string    `json:"endpoint,omitempty" yaml:"endpoint,omitempty"`
	Username string    `json:"username,omitempty" yaml:"username,omitempty"`
	Password string    `json:"password,omitempty" yaml:"password,omitempty"`
	SkipTLS  bool      `json:"skiptls,omitempty" yaml:"skiptls,omitempty"`
	Query    Query     `json:"query,omitempty" yaml:"query,omitempty"`
	Table    MainTable `json:"table,omitempty" yaml:"table,omitempty"`
}

type Query struct {
	BaseDN string  `json:"basedn,omitempty" yaml:"basedn,omitempty"`
	Filter string  `json:"filter,omitempty" yaml:"filter,omitempty"`
	Scope  *string `json:"scope,omitempty" yaml:"scope,omitempty"`
}
