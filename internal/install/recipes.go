package install

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

type EndpointRecipe struct {
	Name          string
	DisplayName   string
	TargetBaseURL string
	Notes         []string
}

type RecipeOptions struct {
	CachyBaseURL string
}

type MCPOptions struct {
	Command       string
	ListenAddress string
	TargetBaseURL string
}

var endpointRecipes = []EndpointRecipe{
	{Name: "ollama", DisplayName: "Ollama", TargetBaseURL: "http://127.0.0.1:11434", Notes: []string{"Ollama exposes an OpenAI-compatible API under /v1."}},
	{Name: "llama.cpp", DisplayName: "llama.cpp server", TargetBaseURL: "http://127.0.0.1:8080", Notes: []string{"Start llama.cpp server with OpenAI-compatible HTTP support."}},
	{Name: "lm-studio", DisplayName: "LM Studio", TargetBaseURL: "http://127.0.0.1:1234", Notes: []string{"Enable LM Studio's local server before starting Cachy."}},
	{Name: "vllm", DisplayName: "vLLM", TargetBaseURL: "http://127.0.0.1:8000", Notes: []string{"Use vLLM's OpenAI-compatible server endpoint."}},
	{Name: "localai", DisplayName: "LocalAI", TargetBaseURL: "http://127.0.0.1:8080", Notes: []string{"Use LocalAI's OpenAI-compatible endpoint."}},
}

func EndpointRecipes() []EndpointRecipe {
	out := make([]EndpointRecipe, len(endpointRecipes))
	copy(out, endpointRecipes)
	return out
}

func EndpointRecipeByName(name string) (EndpointRecipe, error) {
	normalized := strings.ToLower(strings.TrimSpace(name))
	for _, recipe := range endpointRecipes {
		if recipe.Name == normalized {
			return recipe, nil
		}
	}
	return EndpointRecipe{}, fmt.Errorf("unknown endpoint recipe %q", name)
}

func CustomEndpointRecipe(targetBaseURL string) (EndpointRecipe, error) {
	if err := validateBaseURL(targetBaseURL, "custom endpoint target URL"); err != nil {
		return EndpointRecipe{}, err
	}
	return EndpointRecipe{
		Name:          "custom",
		DisplayName:   "Custom OpenAI-compatible endpoint",
		TargetBaseURL: targetBaseURL,
		Notes:         []string{"Use this for any OpenAI-compatible HTTP backend."},
	}, nil
}

func RenderEndpointRecipe(recipe EndpointRecipe, options RecipeOptions) (string, error) {
	cachyBaseURL := strings.TrimRight(options.CachyBaseURL, "/")
	if cachyBaseURL == "" {
		cachyBaseURL = "http://127.0.0.1:8787"
	}
	if err := validateBaseURL(cachyBaseURL, "Cachy base URL"); err != nil {
		return "", err
	}
	if err := validateBaseURL(recipe.TargetBaseURL, "recipe target URL"); err != nil {
		return "", err
	}

	clientBaseURL := cachyBaseURL + "/v1"
	listen := listenAddressFromBaseURL(cachyBaseURL)
	lines := []string{
		"# " + recipe.DisplayName,
		"Start Cachy:",
		fmt.Sprintf("cachy proxy --listen %s --target %s", listen, recipe.TargetBaseURL),
		"",
		"Point OpenAI-compatible clients at Cachy:",
		"OPENAI_BASE_URL=" + clientBaseURL,
		"CACHY_TARGET_BASE_URL=" + recipe.TargetBaseURL,
	}
	for _, note := range recipe.Notes {
		lines = append(lines, "", "Note: "+note)
	}
	return strings.Join(lines, "\n") + "\n", nil
}

func RenderMCPRegistration(options MCPOptions) (string, error) {
	if options.Command == "" {
		options.Command = "cachy"
	}
	if options.ListenAddress == "" {
		options.ListenAddress = "127.0.0.1:8787"
	}
	if err := validateBaseURL(options.TargetBaseURL, "MCP target URL"); err != nil {
		return "", err
	}
	registration := map[string]any{
		"mcpServers": map[string]any{
			"cachy": map[string]any{
				"command": options.Command,
				"args": []string{
					"proxy",
					"--listen", options.ListenAddress,
					"--target", options.TargetBaseURL,
				},
			},
		},
	}
	data, err := json.MarshalIndent(registration, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data) + "\n", nil
}

func listenAddressFromBaseURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return "127.0.0.1:8787"
	}
	return parsed.Host
}
