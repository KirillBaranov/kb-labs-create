package manifest

// Package is a core npm package required by the platform.
type Package struct {
	Name string `json:"name"`
}

// Component is an optional service or plugin.
type Component struct {
	ID          string `json:"id"`
	Pkg         string `json:"pkg"`
	Description string `json:"description"`
	Default     bool   `json:"default"`
}

// Manifest describes all installable parts of the KB Labs platform.
type Manifest struct {
	Version     string      `json:"version"`
	RegistryURL string      `json:"registryUrl"`
	Core        []Package   `json:"core"`
	Services    []Component `json:"services"`
	Plugins     []Component `json:"plugins"`
}

// CorePackageNames returns plain package name strings from Core.
func (m *Manifest) CorePackageNames() []string {
	names := make([]string, len(m.Core))
	for i, p := range m.Core {
		names[i] = p.Name
	}
	return names
}
