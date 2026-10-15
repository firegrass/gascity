package main

import (
	"testing"

	"github.com/gastownhall/gascity/internal/config"
)

func launchModelTestProvider(opt config.ProviderOption, defaults map[string]string) *config.ResolvedProvider {
	return &config.ResolvedProvider{
		Name:              "claude",
		OptionsSchema:     []config.ProviderOption{{Key: "effort", Type: "select", Choices: []config.OptionChoice{{Value: "high", FlagArgs: []string{"--effort", "high"}}}}, opt},
		EffectiveDefaults: defaults,
	}
}

// TestResolvedLaunchModelUsesFlagValue pins that the launch_model stamp is the
// provider-facing id rendered from the selected choice's flag args — never the
// choice key a deployed city pins in option_defaults / template_overrides.
func TestResolvedLaunchModelUsesFlagValue(t *testing.T) {
	closedModel := config.ProviderOption{
		Key:     "model",
		Type:    "select",
		Default: "sonnet-5",
		Choices: []config.OptionChoice{
			{Value: "", Label: "Default"},
			{Value: "opus-5-5", FlagArgs: []string{"--model", "claude-opus-5-5"}, FlagAliases: [][]string{{"-m", "claude-opus-5-5"}}},
			{Value: "sonnet-5", FlagArgs: []string{"--model", "claude-sonnet-5"}},
			{Value: "short-m", FlagArgs: []string{"-m", "claude-haiku-5"}},
			{Value: "assign", FlagArgs: []string{"--model=claude-opus-5-5-assign"}},
			{Value: "codex-style", FlagArgs: []string{"-c", "model=gpt-5.5"}},
		},
	}
	openModel := closedModel
	openModel.FlagTemplate = []string{"--model", config.OptionValuePlaceholder}

	tests := []struct {
		name      string
		rp        *config.ResolvedProvider
		overrides map[string]string
		want      string
	}{
		{
			name:      "override choice key renders its flag value",
			rp:        launchModelTestProvider(closedModel, map[string]string{"model": "sonnet-5"}),
			overrides: map[string]string{"model": "opus-5-5"},
			want:      "claude-opus-5-5",
		},
		{
			name: "effective default when no override",
			rp:   launchModelTestProvider(closedModel, map[string]string{"model": "opus-5-5"}),
			want: "claude-opus-5-5",
		},
		{
			name: "option default when no effective default",
			rp:   launchModelTestProvider(closedModel, nil),
			want: "claude-sonnet-5",
		},
		{
			name:      "override for another option does not select a model",
			rp:        launchModelTestProvider(closedModel, map[string]string{"model": "opus-5-5"}),
			overrides: map[string]string{"effort": "high"},
			want:      "claude-opus-5-5",
		},
		{
			name:      "-m alias flag shape",
			rp:        launchModelTestProvider(closedModel, nil),
			overrides: map[string]string{"model": "short-m"},
			want:      "claude-haiku-5",
		},
		{
			name:      "--model= assignment shape",
			rp:        launchModelTestProvider(closedModel, nil),
			overrides: map[string]string{"model": "assign"},
			want:      "claude-opus-5-5-assign",
		},
		{
			name:      "codex -c assignment falls back to the last token",
			rp:        launchModelTestProvider(closedModel, nil),
			overrides: map[string]string{"model": "codex-style"},
			want:      "model=gpt-5.5",
		},
		{
			name:      "open option renders an undeclared value through its template",
			rp:        launchModelTestProvider(openModel, nil),
			overrides: map[string]string{"model": "claude-opus-6"},
			want:      "claude-opus-6",
		},
		{
			name:      "closed option refuses an undeclared value",
			rp:        launchModelTestProvider(closedModel, nil),
			overrides: map[string]string{"model": "claude-opus-6"},
			want:      "",
		},
		{
			name:      "explicit leave-unset choice pins nothing",
			rp:        launchModelTestProvider(closedModel, nil),
			overrides: map[string]string{"model": ""},
			want:      "claude-sonnet-5",
		},
		{
			name: "provider without a model option",
			rp: &config.ResolvedProvider{Name: "bare", OptionsSchema: []config.ProviderOption{
				{Key: "effort", Type: "select", Choices: []config.OptionChoice{{Value: "high", FlagArgs: []string{"--effort", "high"}}}},
			}, EffectiveDefaults: map[string]string{"model": "opus-5-5"}},
			want: "",
		},
		{
			name: "nil provider",
			rp:   nil,
			want: "",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolvedLaunchModel(tc.rp, tc.overrides); got != tc.want {
				t.Fatalf("resolvedLaunchModel = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestModelFlagValue(t *testing.T) {
	tests := []struct {
		args     []string
		fallback string
		want     string
	}{
		{[]string{"--model", "claude-opus-5-5"}, "x", "claude-opus-5-5"},
		{[]string{"-m", "grok-4.7"}, "x", "grok-4.7"},
		{[]string{"--model=gpt-5.5"}, "x", "gpt-5.5"},
		{[]string{"--effort", "high", "--model", " codex-mini "}, "x", "codex-mini"},
		{[]string{"-c", "model=gpt-5.5"}, "x", "model=gpt-5.5"},
		{[]string{"--model"}, "fallback", "--model"},
		{nil, "fallback", "fallback"},
		{nil, "", ""},
	}
	for _, tc := range tests {
		if got := modelFlagValue(tc.args, tc.fallback); got != tc.want {
			t.Errorf("modelFlagValue(%q, %q) = %q, want %q", tc.args, tc.fallback, got, tc.want)
		}
	}
}
