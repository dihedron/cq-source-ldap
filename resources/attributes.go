package resources

import "github.com/dihedron/cq-source-ldap/client"

type AttributeType int8

const (
	AttributeTypeUnknown AttributeType = iota
	AttributeTypeString
	AttributeTypeInteger
	AttributeTypeBoolean
	AttributeTypeSID
	AttributeTypeStringArray
)

func getAttributeName(c *client.Column) string {
	if c.Attribute != nil {
		return *c.Attribute
	}
	return c.Name
}

func getAttributeType(c *client.Column) AttributeType {
	if c.Type != nil && c.Type.From != nil {
		switch *c.Type.From {
		case "string":
			return AttributeTypeString
		case "integer", "int":
			return AttributeTypeInteger
		case "boolean", "bool":
			return AttributeTypeBoolean
		case "sid":
			return AttributeTypeSID
		case "[]string":
			return AttributeTypeStringArray
		}
	}
	return AttributeTypeUnknown
}
