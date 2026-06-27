package install

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestEndpointRecipesPointClientsThroughCachy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		wantTarget string
	}{
		{name: "ollama", wantTarget: "http://127.0.0.1:11434"},
		{name: "llama.cpp", wantTarget: "http://127.0.0.1:8080"},
		{name: "lm-studio", wantTarget: "http://127.0.0.1:1234"},
		{name: "vllm", wantTarget: "http://127.0.0.1:8000"},
		{name: "localai", wantTarget: "http://127.0.0.1:8080"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			recipe, err := EndpointRecipeByName(tt.name)
			if err != nil {
				t.Fatalf("EndpointRecipeByName() error = %v", err)
			}
			if recipe.TargetBaseURL != tt.wantTarget {
				t.Fatalf("target = %q, want %q", recipe.TargetBaseURL, tt.wantTarget)
			}
			rendered, err := RenderEndpointRecipe(recipe, RecipeOptions{
				CachyBaseURL: "http://127.0.0.1:8787",
			})
			if err != nil {
				t.Fatalf("RenderEndpointRecipe() error = %v", err)
			}
			for _, want := range []string{
				"OPENAI_BASE_URL=http://127.0.0.1:8787/v1",
				"CACHY_TARGET_BASE_URL=" + tt.wantTarget,
				"cachy proxy --listen 127.0.0.1:8787 --target " + tt.wantTarget,
			} {
				if !strings.Contains(rendered, want) {
					t.Fatalf("recipe output missing %q:\n%s", want, rendered)
				}
			}
		})
	}
}

func TestCustomEndpointRecipeValidatesTargetURL(t *testing.T) {
	t.Parallel()

	recipe, err := CustomEndpointRecipe("http://10.0.0.5:9000")
	if err != nil {
		t.Fatalf("CustomEndpointRecipe() error = %v", err)
	}
	if recipe.Name != "custom" || recipe.TargetBaseURL != "http://10.0.0.5:9000" {
		t.Fatalf("recipe = %#v, want custom target", recipe)
	}
	if _, err := CustomEndpointRecipe("not-a-url"); err == nil {
		t.Fatal("CustomEndpointRecipe() invalid URL error = nil, want error")
	}
}

func TestRenderMCPRegistrationUsesCachyProxyCommand(t *testing.T) {
	t.Parallel()

	registration, err := RenderMCPRegistration(MCPOptions{
		Command:       "cachy",
		ListenAddress: "127.0.0.1:8787",
		TargetBaseURL: "http://127.0.0.1:11434",
	})
	if err != nil {
		t.Fatalf("RenderMCPRegistration() error = %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(registration), &parsed); err != nil {
		t.Fatalf("registration is not JSON: %v\n%s", err, registration)
	}
	server := parsed["mcpServers"].(map[string]any)["cachy"].(map[string]any)
	if server["command"] != "cachy" {
		t.Fatalf("command = %#v, want cachy", server["command"])
	}
	args := server["args"].([]any)
	joined := strings.Trim(strings.Join(anyStrings(args), " "), " ")
	for _, want := range []string{"proxy", "--listen 127.0.0.1:8787", "--target http://127.0.0.1:11434"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("args = %q, missing %q", joined, want)
		}
	}
}

func anyStrings(values []any) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, value.(string))
	}
	return out
}
