package config

// ModelDef defines a supported model's static configuration.
type ModelDef struct {
	Name             string
	APIURL           string
	MaxContextTokens int
}

// SupportedModels is the hardcoded list of models the binary supports.
var SupportedModels = []ModelDef{
	{
		Name:             "glm-4.5-air",
		APIURL:           "https://api.z.ai/api/paas/v4/chat/completions",
		MaxContextTokens: 128000,
	},
}

// ModelByName returns the model definition for the given name, or nil if not found.
func ModelByName(name string) *ModelDef {
	for i := range SupportedModels {
		if SupportedModels[i].Name == name {
			return &SupportedModels[i]
		}
	}
	return nil
}

// ModelNames returns the names of all supported models.
func ModelNames() []string {
	names := make([]string, len(SupportedModels))
	for i, m := range SupportedModels {
		names[i] = m.Name
	}
	return names
}
