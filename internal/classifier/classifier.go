package classifier

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"charm.land/fantasy"
	"github.com/charmbracelet/crush/internal/config"
)

// ClassifyRequest holds the information needed to classify a tool call.
type ClassifyRequest struct {
	ToolName   string
	Action     string
	Params     any
	Path       string
	WorkingDir string
}

// ClassifyResult holds the outcome of a classification.
type ClassifyResult struct {
	Verdict Verdict
	Reason  string
	// Stage indicates which classification stage produced the result.
	// 0 = static rules, 1 = Stage 1 fast filter, 2 = Stage 2 CoT.
	Stage int
}

// Service classifies tool calls for auto-mode permission decisions.
type Service struct {
	model      fantasy.LanguageModel
	autoConfig config.AutoModeConfig
	timeout    time.Duration

	mu                 sync.Mutex
	consecutiveDenials int
	totalDenials       int
	paused             bool
}

// NewService creates a classifier service. If model is nil, the service
// operates in rules-only mode (no LLM classification; ambiguous calls
// escalate to the user).
func NewService(model fantasy.LanguageModel, autoConfig config.AutoModeConfig) *Service {
	cfg := autoConfig.Defaults()
	return &Service{
		model:      model,
		autoConfig: cfg,
		timeout:    time.Duration(cfg.ClassifierTimeout) * time.Second,
	}
}

// Paused reports whether the classifier has hit the denial circuit
// breaker and fallen back to manual mode.
func (s *Service) Paused() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.paused
}

// RecordApproval resets the consecutive denial counter on a successful
// approval (from either the classifier or the user).
func (s *Service) RecordApproval() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.consecutiveDenials = 0
}

// recordDenial increments denial counters and returns true if the
// circuit breaker has tripped.
func (s *Service) recordDenial() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.consecutiveDenials++
	s.totalDenials++
	if s.consecutiveDenials >= s.autoConfig.MaxConsecutiveDenials ||
		s.totalDenials >= s.autoConfig.MaxTotalDenials {
		s.paused = true
		return true
	}
	return false
}

// Classify evaluates a tool call through the static rules and, if
// needed, the LLM classifier. Returns a ClassifyResult with the
// verdict and optional reason.
func (s *Service) Classify(ctx context.Context, req ClassifyRequest) ClassifyResult {
	if s.Paused() {
		return ClassifyResult{
			Verdict: VerdictEscalate,
			Reason:  "auto mode paused: denial limit reached",
		}
	}

	trustWrites := s.autoConfig.TrustProjectWrites != nil && *s.autoConfig.TrustProjectWrites

	verdict := ClassifyByRules(req.ToolName, req.Action, req.Path, req.WorkingDir, trustWrites)
	if verdict != VerdictClassify {
		if verdict == VerdictAllow {
			s.RecordApproval()
		}
		return ClassifyResult{Verdict: verdict, Stage: 0}
	}

	if s.model == nil {
		return ClassifyResult{
			Verdict: VerdictEscalate,
			Reason:  "no classifier model configured",
		}
	}

	paramsJSON := marshalParams(req.Params)

	result := s.runClassifier(ctx, req.WorkingDir, req.ToolName, req.Action, paramsJSON)
	if result.Verdict == VerdictDeny {
		tripped := s.recordDenial()
		if tripped {
			slog.Warn("Auto mode circuit breaker tripped",
				"consecutive", s.consecutiveDenials,
				"total", s.totalDenials)
		}
	} else if result.Verdict == VerdictAllow {
		s.RecordApproval()
	}
	return result
}

// runClassifier executes the two-stage LLM classification pipeline.
func (s *Service) runClassifier(ctx context.Context, workingDir, toolName, action, paramsJSON string) ClassifyResult {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	stage1Output, err := s.generate(ctx, Stage1Prompt(workingDir, toolName, action, paramsJSON))
	if err != nil {
		slog.Warn("Classifier Stage 1 failed, escalating to user",
			"error", err, "tool", toolName)
		return ClassifyResult{
			Verdict: VerdictEscalate,
			Reason:  fmt.Sprintf("classifier unavailable: %s", err),
			Stage:   1,
		}
	}

	stage1Verdict := ParseStage1(stage1Output)
	if stage1Verdict == VerdictAllow {
		return ClassifyResult{Verdict: VerdictAllow, Stage: 1}
	}

	stage2Output, err := s.generate(ctx, Stage2Prompt(workingDir, toolName, action, paramsJSON, stage1Verdict.String()))
	if err != nil {
		slog.Warn("Classifier Stage 2 failed, escalating to user",
			"error", err, "tool", toolName)
		return ClassifyResult{
			Verdict: VerdictEscalate,
			Reason:  fmt.Sprintf("classifier Stage 2 unavailable: %s", err),
			Stage:   2,
		}
	}

	stage2 := ParseStage2(stage2Output)
	return ClassifyResult{
		Verdict: stage2.Verdict,
		Reason:  stage2.Reason,
		Stage:   2,
	}
}

// generate sends a simple single-turn prompt to the classifier model
// and returns the text response.
func (s *Service) generate(ctx context.Context, prompt string) (string, error) {
	call := fantasy.Call{
		Prompt: fantasy.Prompt{
			{
				Role: fantasy.MessageRoleUser,
				Content: []fantasy.MessagePart{
					fantasy.TextPart{Text: prompt},
				},
			},
		},
		MaxOutputTokens: int64Ptr(256),
		Temperature:     float64Ptr(0),
	}

	resp, err := s.model.Generate(ctx, call)
	if err != nil {
		return "", err
	}

	return resp.Content.Text(), nil
}

func marshalParams(params any) string {
	if params == nil {
		return "{}"
	}
	data, err := json.Marshal(params)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func float64Ptr(v float64) *float64 {
	return &v
}

func int64Ptr(v int64) *int64 {
	return &v
}
