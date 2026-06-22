package synthesis

import (
	"context"
	"fmt"
	"os"
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

// ValidatePost evaluates a crafted post against strict ecosystem rules.
func ValidatePost(text string) error {
	text = strings.TrimSpace(text)

	// 1. Max 20 words
	words := strings.Fields(text)
	if len(words) > 20 {
		return fmt.Errorf("post exceeds 20 words limit (has %d words)", len(words))
	}

	// 2. 0% instances of first-person pronouns
	bannedPronouns := map[string]bool{
		"i": true, "me": true, "my": true, "we": true, "our": true, "us": true,
	}
	for _, w := range words {
		cleaned := strings.Trim(strings.ToLower(w), ".,;:!?()[]{}'\"`*#_")
		if bannedPronouns[cleaned] {
			return fmt.Errorf("post contains banned first-person pronoun: %q", w)
		}
	}

	// 3. 0% instances of actual URLs/hyperlinks
	lowerText := strings.ToLower(text)
	if strings.Contains(lowerText, "http://") || strings.Contains(lowerText, "https://") || strings.Contains(lowerText, "www.") {
		return fmt.Errorf("post contains a URL or link pattern")
	}

	// 4. Plain alphanumeric string. 0% Markdown tokens (*, _, #, `, etc).
	markdownTokens := []string{"*", "_", "#", "`", "[", "]", "(", ")", "~", ">"}
	for _, tok := range markdownTokens {
		if strings.Contains(text, tok) {
			return fmt.Errorf("post contains markdown token: %q", tok)
		}
	}

	// 5. Max 1 emoji. 0% exclamation marks (!).
	if strings.Contains(text, "!") {
		return fmt.Errorf("post contains exclamation mark")
	}

	emojiCount := 0
	for _, r := range text {
		// Basic range check for emojis
		if (r >= 0x1F300 && r <= 0x1F9FF) || (r >= 0x2600 && r <= 0x26FF) || (r >= 0x2700 && r <= 0x27BF) {
			emojiCount++
		}
	}
	if emojiCount > 1 {
		return fmt.Errorf("post contains more than 1 emoji (has %d)", emojiCount)
	}

	// 6. Banned tokens (AI ticks & Marketing cliches):
	bannedTokens := []string{
		"delve", "revolutionize", "game-changer", "seamless", "testament", "unlock", "empower", "streamline", "landscape", "elevate", "next-level", "next-gen",
		"excited to announce", "look no further", "transforming the way", "are you tired of", "look at this", "introducing",
	}
	for _, tok := range bannedTokens {
		if strings.Contains(lowerText, tok) {
			return fmt.Errorf("post contains banned marketing/AI token: %q", tok)
		}
	}

	return nil
}

func GetProjectManifest(p *project.Project) string {
	if p.ManifestPath != "" {
		if data, err := os.ReadFile(p.ManifestPath); err == nil {
			return string(data)
		}
	}
	return p.Description
}

func (s *Synthesizer) CraftPost(ctx context.Context, p *project.Project, assets []*media.Asset, goal string, cta string, recentCopies []string) (string, error) {
	manifest := GetProjectManifest(p)

	var mediaContext strings.Builder
	for i, a := range assets {
		if a.OCRText != "" {
			mediaContext.WriteString(fmt.Sprintf("Asset %d (OCR Content): %s\n", i+1, a.OCRText))
		}
		if a.AISummary != "" {
			mediaContext.WriteString(fmt.Sprintf("Asset %d (Visual Summary): %s\n", i+1, a.AISummary))
		}
	}

	// Retry loop for the LLM output to satisfy constraints
	var lastErr error
	var candidatePost string

	for attempt := 1; attempt <= 5; attempt++ {
		feedbackPrompt := ""
		if attempt > 1 {
			feedbackPrompt = fmt.Sprintf("\n\n### CRITICAL CORRECTION\nYour previous attempt: %q failed validation: %v.\nRegenerate making sure to fix this error.", candidatePost, lastErr)
		}

		avoidPrompt := ""
		if len(recentCopies) > 0 {
			avoidPrompt = fmt.Sprintf("\n\n### CRITICAL VARIATION CONSTRAINT\nDo NOT generate any text similar to these previous copies:\n- %s", strings.Join(recentCopies, "\n- "))
		}

		prompt := fmt.Sprintf(`### ROLE
You are an autonomous technical protocol generator. You speak in a highly dense, absolute objective tone.
You are generating a post about the project: %s.

### CONTEXT
Product Manifest / Architecture:
%s

### MANDATE
1. Derive context exclusively from the provided Manifest/Architecture. If a feature or protocol is not listed, do not reference it.
2. Character/Word limit: Write a statement of at most 8 to 10 words. Keep it extremely brief.
3. Pronoun filter: Do NOT use any first-person pronouns (I, me, my, we, our, us).
4. Links: Do NOT include any links, URLs, or website domains.
5. Markdown: Output raw text only. Do not use any markdown tokens (no *, _, #, %s, etc).
6. Exclamation: Never use exclamation marks.
7. Tone: Objective, technical, descriptive. Do not sound like a marketer, human creator, or indie developer. Do not refer to ownership.
8. Banned Words: Do not use: delve, revolutionize, game-changer, seamless, testament, unlock, empower, streamline, landscape, elevate, next-level, next-gen, excited to announce, look no further, transforming the way, are you tired of, look at this, introducing.

### TASK
Goal: %s
Media Context: %s
%s

Output ONLY the raw statement without any CTA or links.%s`, p.Name, manifest, "`", goal, mediaContext.String(), avoidPrompt, feedbackPrompt)

		intent := p.GenerationMode
		if intent == "" {
			intent = "completion"
		}
		resp, err := s.AI.Query(prompt, intent, "github-models", "")
		if err != nil {
			return "", err
		}

		// Remove quotes, if any
		candidatePost = strings.Trim(strings.TrimSpace(resp), "\"`")

		// Combine with CTA
		fullPost := candidatePost
		if cta != "" {
			fullPost = candidatePost + " " + cta
		}

		// Validate
		if err := ValidatePost(fullPost); err == nil {
			return fullPost, nil
		} else {
			lastErr = err
		}
	}

	// If all retries failed, return a fallback post that is guaranteed valid
	fallback := fmt.Sprintf("Technical framework operates autonomously. %s", cta)
	if err := ValidatePost(fallback); err == nil {
		return fallback, nil
	}

	return fmt.Sprintf("Protocol active. %s", cta), nil
}

func (s *Synthesizer) CraftReply(ctx context.Context, p *project.Project, threadContent string, assets []*media.Asset, recentCopies []string) (string, error) {
	manifest := GetProjectManifest(p)

	var mediaContext strings.Builder
	for i, a := range assets {
		if a.OCRText != "" {
			mediaContext.WriteString(fmt.Sprintf("My Asset %d (OCR): %s\n", i+1, a.OCRText))
		}
	}

	avoidPrompt := ""
	if len(recentCopies) > 0 {
		avoidPrompt = fmt.Sprintf("\n\n### CRITICAL VARIATION CONSTRAINT\nDo NOT generate any text similar to these previous copies:\n- %s", strings.Join(recentCopies, "\n- "))
	}

	prompt := fmt.Sprintf(`### ROLE
You are an autonomous technical protocol generator. You speak in a highly dense, absolute objective tone.

### CONTEXT
Product Manifest / Architecture:
%s

### THREAD TO REPLY TO
%s

### MY ASSETS
%s

### MANDATE
1. Craft a response under 15 words.
2. 0%% first-person pronouns (I, me, my, we, our, us).
3. Plain alphanumeric string. 0%% Markdown tokens.
4. Max 1 emoji. 0%% exclamation marks.
5. Absolute objective tone.
6. Banned words: delve, revolutionize, game-changer, seamless, testament, unlock, empower, streamline, landscape, elevate, next-level, next-gen.
%s

Raw reply text only.`, manifest, threadContent, mediaContext.String(), avoidPrompt)

	intent := p.GenerationMode
	if intent == "" {
		intent = "completion"
	}
	return s.AI.Query(prompt, intent, "github-models", "")
}
