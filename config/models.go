package config

// ModelDef defines a supported model's static configuration.
type ModelDef struct {
	Name             string
	APIURL           string
	MaxContextTokens int
}

const zaiEndpoint = "https://api.z.ai/api/paas/v4/chat/completions"

// SupportedModels is the hardcoded list of models the binary supports.
var SupportedModels = []ModelDef{
	// GLM-4.7 series — 200K context
	{Name: "glm-4.7", APIURL: zaiEndpoint, MaxContextTokens: 200000},
	{Name: "glm-4.7-flashx", APIURL: zaiEndpoint, MaxContextTokens: 200000},
	{Name: "glm-4.7-flash", APIURL: zaiEndpoint, MaxContextTokens: 200000},
	// GLM-4.5 series — 128K context
	{Name: "glm-4.5", APIURL: zaiEndpoint, MaxContextTokens: 128000},
	{Name: "glm-4.5-air", APIURL: zaiEndpoint, MaxContextTokens: 128000},
	{Name: "glm-4.5-x", APIURL: zaiEndpoint, MaxContextTokens: 128000},
	{Name: "glm-4.5-airx", APIURL: zaiEndpoint, MaxContextTokens: 128000},
	{Name: "glm-4.5-flash", APIURL: zaiEndpoint, MaxContextTokens: 128000},
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
