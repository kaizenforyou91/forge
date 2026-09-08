package tool

import "strings"

// NewCatalog validates and copies all definitions before returning a catalog.
// Names are 1..64 ASCII bytes matching [A-Za-z_][A-Za-z0-9_-]*.
// At most 64 tools and 32 parameters per tool are permitted. Duplicates,
// including identical duplicates, invalidate the entire construction.
// Nil/empty input is valid and denies every tool. No I/O is performed.
// Callers must not mutate input slices while construction is reading them;
// mutation after return cannot change the catalog.
func NewCatalog(definitions []Definition) (*Catalog, error) {
	if len(definitions) > maxTools {
		return nil, ErrInvalidCatalog
	}
	index := make(map[string]definition, len(definitions))
	for _, candidate := range definitions {
		if !validName(candidate.Name) || len(candidate.Parameters) > maxParameters {
			return nil, ErrInvalidCatalog
		}
		if _, exists := index[candidate.Name]; exists {
			return nil, ErrInvalidCatalog
		}
		parameters := make(map[string]Parameter, len(candidate.Parameters))
		for _, parameter := range candidate.Parameters {
			if !validName(parameter.Name) || parameter.Type < String || parameter.Type > Null {
				return nil, ErrInvalidCatalog
			}
			if _, exists := parameters[parameter.Name]; exists {
				return nil, ErrInvalidCatalog
			}
			parameter.Name = strings.Clone(parameter.Name)
			parameters[parameter.Name] = parameter
		}
		index[strings.Clone(candidate.Name)] = definition{parameters: parameters}
	}
	return &Catalog{definitions: index}, nil
}

func validName(name string) bool {
	if len(name) == 0 || len(name) > maxNameBytes {
		return false
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		initial := c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c == '_'
		if !initial && (i == 0 || !(c >= '0' && c <= '9' || c == '-')) {
			return false
		}
	}
	return true
}
