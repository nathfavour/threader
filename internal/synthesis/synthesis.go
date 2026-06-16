package synthesis

import (
	"context"
	"fmt"
	"strings"

	"github.com/nathfavour/threader/internal/ai"
	"github.com/nathfavour/threader/internal/media"
	"github.com/nathfavour/threader/internal/project"
)

type Synthesizer struct {
	AI *ai.Client
}

func NewSynthesizer(aiClient *ai.Client) *Synthesizer {
	return &Synthesizer{AI: aiClient}
}

func (s *Synthesizer) CraftPost(ctx context.Context, p *project.Project, assets []*media.Asset, goal string) (string, error) {
	var mediaContext strings.Builder
	for i, a := range assets {
		if a.OCRText != "" {
			mediaContext.WriteString(fmt.Sprintf("Asset %d (OCR Content): %s\n", i+1, a.OCRText))
		}
		if a.AISummary != "" {
			mediaContext.WriteString(fmt.Sprintf("Asset %d (Visual Summary): %s\n", i+1, a.AISummary))
		}
	}

	prompt := fmt.Sprintf(
		"%s\n\n### PROJECT CONTEXT\nName: %s\nDescription: %s\nBrand Voice: %s\nWebsite: %s\nCodebase: %s\n\n### TASK\nGoal: %s\nMedia Context: %s\n\n### INSTRUCTION\nCraft a post following the Blueprint rules above. Be extremely concise. Use the media content to inform your writing but do not describe it. Final output should be the raw post text only.",
		MarketingBlueprint, p.Name, p.Description, p.BrandVoice, p.WebsiteURL, p.CodebaseURL, goal, mediaContext.String(),
	)

	return s.AI.Query(prompt, "vibe", "github-models", "")
}

func (s *Synthesizer) CraftReply(ctx context.Context, p *project.Project, threadContent string, assets []*media.Asset) (string, error) {
	var mediaContext strings.Builder
	for i, a := range assets {
		if a.OCRText != "" {
			mediaContext.WriteString(fmt.Sprintf("My Asset %d (OCR): %s\n", i+1, a.OCRText))
		}
	}

	prompt := fmt.Sprintf(
		"%s\n\n### PROJECT CONTEXT\nName: %s\nBrand Voice: %s\nWebsite: %s\nCodebase: %s\n\n### THREAD TO REPLY TO\n%s\n\n### MY ASSETS\n%s\n\n### INSTRUCTION\nCraft a human-like reply following the Blueprint rules. Be high-signal and low-noise. Do not market. Use my assets ONLY if they provide value to the conversation. Redirect to profile for more info. Raw reply text only.",
		MarketingBlueprint, p.Name, p.BrandVoice, p.WebsiteURL, p.CodebaseURL, threadContent, mediaContext.String(),
	)

	return s.AI.Query(prompt, "vibe", "github-models", "")
}
