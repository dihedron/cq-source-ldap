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
		return c.Attribute.Name
	}
	return c.Name
}

func getAttributeType(c *client.Column) AttributeType {
	if c.Attribute != nil && c.Attribute.Type != nil {
		switch *c.Attribute.Type {
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
