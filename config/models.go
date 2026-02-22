package config

// ModelDef defines a supported model's static configuration.
type ModelDef struct {
	Name             string
	APIURL           string
	MaxContextTokens int
	APIKeyName       string // key used to look up the API key in settings (e.g. "Z_API")
}

const zaiEndpoint = "https://api.z.ai/api/paas/v4/chat/completions"

// SupportedModels is the hardcoded list of models the binary supports.
var SupportedModels = []ModelDef{
	// GLM-5 series — 200K context
	{Name: "glm-5", APIURL: zaiEndpoint, MaxContextTokens: 200000, APIKeyName: "Z_API"},
	// GLM-4.7 series — 200K context
	{Name: "glm-4.7", APIURL: zaiEndpoint, MaxContextTokens: 200000, APIKeyName: "Z_API"},
	{Name: "glm-4.7-flashx", APIURL: zaiEndpoint, MaxContextTokens: 200000, APIKeyName: "Z_API"},
	{Name: "glm-4.7-flash", APIURL: zaiEndpoint, MaxContextTokens: 200000, APIKeyName: "Z_API"},
	// GLM-4.5 series — 128K context
	{Name: "glm-4.5", APIURL: zaiEndpoint, MaxContextTokens: 128000, APIKeyName: "Z_API"},
	{Name: "glm-4.5-air", APIURL: zaiEndpoint, MaxContextTokens: 128000, APIKeyName: "Z_API"},
	{Name: "glm-4.5-x", APIURL: zaiEndpoint, MaxContextTokens: 128000, APIKeyName: "Z_API"},
	{Name: "glm-4.5-airx", APIURL: zaiEndpoint, MaxContextTokens: 128000, APIKeyName: "Z_API"},
	{Name: "glm-4.5-flash", APIURL: zaiEndpoint, MaxContextTokens: 128000, APIKeyName: "Z_API"},
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
