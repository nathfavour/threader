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
			mediaContext.WriteString(fmt.Sprintf("Asset %d (OCR): %s\n", i+1, a.OCRText))
		}
		if a.AISummary != "" {
			mediaContext.WriteString(fmt.Sprintf("Asset %d (Visual): %s\n", i+1, a.AISummary))
		}
	}

	prompt := fmt.Sprintf(
		"### PROJECT CONTEXT\nName: %s\nDescription: %s\nBrand Voice: %s\n\n### GOAL\n%s\n\n### MEDIA ASSETS\n%s\n\n### INSTRUCTION\nCraft a high-quality, human-sounding post for Threads (Meta's social platform). The post should sound like it was written by the brand's social media manager. Use the media context to make it specific and engaging. Avoid robotic language or excessive hashtags.",
		p.Name, p.Description, p.BrandVoice, goal, mediaContext.String(),
	)

	return s.AI.Query(prompt, "vibe")
}
