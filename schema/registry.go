package schema

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"unicode/utf8"

	"github.com/MeGaNeKoS/neoma/core"
)

// RegistryConfig controls schema generation behavior for a registry.
type RegistryConfig struct {
	AllowAdditionalPropertiesByDefault bool
	FieldsOptionalByDefault            bool
}

// ConfigProvider is implemented by registries that expose their configuration.
type ConfigProvider interface {
	RegistryConfig() RegistryConfig
}

const schemaRefPrefix = "#/components/schemas/"

// MapRegistry is the default core.Registry implementation that stores schemas
// in a map keyed by type name. It handles schema generation, caching, and
// reference resolution for OpenAPI components.
type MapRegistry struct {
	schemas map[string]*core.Schema
	types   map[string]reflect.Type
	seen    map[reflect.Type]bool
	namer   func(reflect.Type, string) string
	aliases map[reflect.Type]reflect.Type
	config  RegistryConfig
}

// Schema returns the schema for the given type, generating and caching it if
// needed. When allowRef is true and the type is a named struct, a $ref schema
// is returned instead of the full definition.
func (r *MapRegistry) Schema(t reflect.Type, allowRef bool, hint string) *core.Schema {
	origType := t
	t = deref(t)

	// Pointer to array should decay to array.
	if t.Kind() == reflect.Array || t.Kind() == reflect.Slice {
		origType = t
	}

	alias, ok := r.aliases[t]
	if ok {
		return r.Schema(alias, allowRef, hint)
	}

	getsRef := t.Kind() == reflect.Struct
	if t == timeType {
		// Special case: time.Time is always a string.
		getsRef = false
	}

	if getsRef {
		ptrT := reflect.PointerTo(t)
		if t.Implements(schemaProviderType) || ptrT.Implements(schemaProviderType) {
			// Special case: type provides its own schema.
			getsRef = false
		} else if t.Implements(textUnmarshalerType) || ptrT.Implements(textUnmarshalerType) {
			// Special case: type can be unmarshaled from text so will be a string
			// and doesn't need a ref. This simplifies the schema a little bit.
			getsRef = false
		}
	}

	name := r.namer(origType, hint)

	if getsRef {
		if s, ok := r.schemas[name]; ok {
			if _, ok = r.seen[t]; !ok {
				// The name matches but the type is different, so we have a dupe.
				panic(fmt.Errorf("duplicate name: %s, new type: %s, existing type: %s", name, t, r.types[name]))
			}

			if allowRef {
				return &core.Schema{Ref: schemaRefPrefix + name}
			}

			return s
		}
	}

	// First, register the type so refs can be created above for recursive types.
	if getsRef {
		r.schemas[name] = &core.Schema{}
		r.types[name] = t
		r.seen[t] = true
	}

	s := FromType(r, origType)
	if getsRef {
		r.schemas[name] = s
	}

	if getsRef && allowRef {
		return &core.Schema{Ref: schemaRefPrefix + name}
	}
	return s
}

// SchemaFromRef resolves a $ref string to its corresponding schema, or returns
// nil if the ref does not match a known schema.
func (r *MapRegistry) SchemaFromRef(ref string) *core.Schema {
	if !strings.HasPrefix(ref, schemaRefPrefix) {
		return nil
	}
	return r.schemas[ref[len(schemaRefPrefix):]]
}

// TypeFromRef returns the Go type associated with the given $ref string.
func (r *MapRegistry) TypeFromRef(ref string) reflect.Type {
	return r.types[ref[len(schemaRefPrefix):]]
}

// Map returns the underlying map of schema names to schema definitions.
func (r *MapRegistry) Map() map[string]*core.Schema {
	return r.schemas
}

// MarshalJSON serializes the registry's schemas as JSON.
func (r *MapRegistry) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.schemas)
}

// MarshalYAML serializes the registry's schemas as YAML.
func (r *MapRegistry) MarshalYAML() (any, error) {
	return r.schemas, nil
}

// RegisterTypeAlias registers an alias so that schemas requested for type t
// are generated using the alias type instead.
func (r *MapRegistry) RegisterTypeAlias(t reflect.Type, alias reflect.Type) {
	r.aliases[t] = alias
}

// RegistryConfig returns the configuration for this registry.
func (r *MapRegistry) RegistryConfig() RegistryConfig {
	return r.config
}

// DefaultSchemaNamer generates a human-readable schema name from a Go type.
// It strips package paths, uppercases the first letter, and handles generics
// and slices (e.g. []int becomes ListInt).
func DefaultSchemaNamer(t reflect.Type, hint string) string {
	name := deref(t).Name()

	if name == "" {
		name = hint
	}

	// Better support for lists, so e.g. []int becomes ListInt.
	name = strings.ReplaceAll(name, "[]", "List[")

	var result strings.Builder
	for _, part := range strings.FieldsFunc(name, func(r rune) bool {
		// Split on special characters. Note that, is used when there are
		// multiple inputs to a generic type.
		return r == '[' || r == ']' || r == '*' || r == ','
	}) {
		// Split fully qualified names like github.com/foo/bar.Baz into Baz.
		lastDot := strings.LastIndex(part, ".")
		base := part[lastDot+1:]

		// Add to result and uppercase for better scalar support (int -> Int).
		// Use unicode-aware uppercase to support non-ASCII characters.
		if len(base) > 0 {
			r, size := utf8.DecodeRuneInString(base)
			if r < utf8.RuneSelf && 'a' <= r && r <= 'z' {
				result.WriteByte(byte(r - ('a' - 'A')))
				result.WriteString(base[size:])
			} else {
				result.WriteString(strings.ToUpper(string(r)))
				result.WriteString(base[size:])
			}
		}
	}

	return result.String()
}

// NewMapRegistry creates a new MapRegistry with the given naming function.
// Use DefaultSchemaNamer for the standard naming convention.
func NewMapRegistry(namer func(t reflect.Type, hint string) string) *MapRegistry {
	return &MapRegistry{
		schemas: map[string]*core.Schema{},
		types:   map[string]reflect.Type{},
		seen:    map[reflect.Type]bool{},
		aliases: map[reflect.Type]reflect.Type{},
		namer:   namer,
		config: RegistryConfig{
			AllowAdditionalPropertiesByDefault: false,
			FieldsOptionalByDefault:            false,
		},
	}
}

// NewMapRegistryWithConfig creates a new MapRegistry with the given naming
// function and configuration overrides.
func NewMapRegistryWithConfig(namer func(t reflect.Type, hint string) string, config RegistryConfig) *MapRegistry {
	r := NewMapRegistry(namer)
	r.config = config
	return r
}

func getRegistryConfig(r core.Registry) RegistryConfig {
	if cp, ok := r.(ConfigProvider); ok {
		return cp.RegistryConfig()
	}
	return RegistryConfig{}
}
