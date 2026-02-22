package config

// ModelDef defines a supported model's static configuration.
type ModelDef struct {
	Name             string
	APIURL           string
	MaxContextTokens int
	APIKeyName       string // key used to look up the API key in settings (e.g. "Z_API")
}

// SupportedModels is the hardcoded list of models the binary supports.
var SupportedModels = []ModelDef{
	// Z.AI Provider
	{Name: "glm-5", APIURL: "https://api.z.ai/api/paas/v4/chat/completions", MaxContextTokens: 200000, APIKeyName: "Z_API"},
	{Name: "glm-4.7", APIURL: "https://api.z.ai/api/paas/v4/chat/completions", MaxContextTokens: 200000, APIKeyName: "Z_API"},
	{Name: "glm-4.7-flashx", APIURL: "https://api.z.ai/api/paas/v4/chat/completions", MaxContextTokens: 200000, APIKeyName: "Z_API"},
	{Name: "glm-4.7-flash", APIURL: "https://api.z.ai/api/paas/v4/chat/completions", MaxContextTokens: 200000, APIKeyName: "Z_API"},
	{Name: "glm-4.5", APIURL: "https://api.z.ai/api/paas/v4/chat/completions", MaxContextTokens: 128000, APIKeyName: "Z_API"},
	{Name: "glm-4.5-air", APIURL: "https://api.z.ai/api/paas/v4/chat/completions", MaxContextTokens: 128000, APIKeyName: "Z_API"},
	{Name: "glm-4.5-x", APIURL: "https://api.z.ai/api/paas/v4/chat/completions", MaxContextTokens: 128000, APIKeyName: "Z_API"},
	{Name: "glm-4.5-airx", APIURL: "https://api.z.ai/api/paas/v4/chat/completions", MaxContextTokens: 128000, APIKeyName: "Z_API"},
	{Name: "glm-4.5-flash", APIURL: "https://api.z.ai/api/paas/v4/chat/completions", MaxContextTokens: 128000, APIKeyName: "Z_API"},
	
	// OpenAI Compatible Models
	{Name: "gpt-4o", APIURL: "https://api.openai.com/v1/chat/completions", MaxContextTokens: 128000, APIKeyName: "OPENAI_API"},
	{Name: "gpt-4o-mini", APIURL: "https://api.openai.com/v1/chat/completions", MaxContextTokens: 128000, APIKeyName: "OPENAI_API"},
	{Name: "gpt-4-turbo", APIURL: "https://api.openai.com/v1/chat/completions", MaxContextTokens: 128000, APIKeyName: "OPENAI_API"},
	{Name: "gpt-3.5-turbo", APIURL: "https://api.openai.com/v1/chat/completions", MaxContextTokens: 16384, APIKeyName: "OPENAI_API"},
	{Name: "codex-5.3", APIURL: "https://api.openai.com/v1/chat/completions", MaxContextTokens: 128000, APIKeyName: "OPENAI_API"},
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
