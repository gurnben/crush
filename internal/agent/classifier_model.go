package agent

import (
	"context"
	"log/slog"

	"charm.land/fantasy"
	"github.com/charmbracelet/crush/internal/config"
)

// BuildClassifierModel builds a fantasy.LanguageModel for the auto-mode
// classifier from the resolved classifier model config. Returns nil if
// the classifier model is not configured or cannot be built.
func BuildClassifierModel(ctx context.Context, store *config.ConfigStore) fantasy.LanguageModel {
	cfg := store.Config()
	classifierCfg, ok := cfg.Models[config.SelectedModelTypeClassifier]
	if !ok || classifierCfg.Model == "" || classifierCfg.Provider == "" {
		slog.Warn("No classifier model configured for auto mode")
		return nil
	}

	providerCfg, ok := cfg.Providers.Get(classifierCfg.Provider)
	if !ok {
		slog.Warn("Classifier model provider not found",
			"provider", classifierCfg.Provider)
		return nil
	}

	// Build a minimal coordinator just to reuse the provider-building
	// logic. This is a temporary coordinator instance that only lives
	// long enough to build the provider.
	c := &coordinator{cfg: store}

	provider, err := c.buildProvider(providerCfg, classifierCfg, true)
	if err != nil {
		slog.Warn("Failed to build classifier provider",
			"error", err, "provider", classifierCfg.Provider)
		return nil
	}

	model, err := provider.LanguageModel(ctx, classifierCfg.Model)
	if err != nil {
		slog.Warn("Failed to build classifier language model",
			"error", err, "model", classifierCfg.Model)
		return nil
	}

	slog.Info("Built classifier model for auto mode",
		"provider", classifierCfg.Provider,
		"model", classifierCfg.Model)
	return model
}
