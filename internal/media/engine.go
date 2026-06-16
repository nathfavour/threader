package media

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/otiai10/gosseract/v2"
)

type Asset struct {
	ID         string    `json:"id"`
	ProjectID  string    `json:"project_id"`
	FilePath   string    `json:"file_path"`
	OCRText    string    `json:"ocr_text"`
	AISummary  string    `json:"ai_summary"`
	UploadedAt time.Time `json:"uploaded_at"`
}

type Engine struct {
	BaseDir string
}

func NewEngine(baseDir string) *Engine {
	return &Engine{BaseDir: baseDir}
}

func (e *Engine) IndexMedia(projectID, srcPath string) (*Asset, error) {
	// 1. Create project media directory
	projectDir := filepath.Join(e.BaseDir, projectID, "media")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return nil, err
	}

	// 2. Copy file to project directory
	assetID := uuid.New().String()
	ext := filepath.Ext(srcPath)
	destPath := filepath.Join(projectDir, assetID+ext)

	data, err := os.ReadFile(srcPath)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(destPath, data, 0644); err != nil {
		return nil, err
	}

	// 3. Perform OCR
	client := gosseract.NewClient()
	defer client.Close()

	if err := client.SetImage(destPath); err != nil {
		// If OCR fails (e.g. not an image), we still keep the asset
		return &Asset{
			ID:         assetID,
			ProjectID:  projectID,
			FilePath:   destPath,
			UploadedAt: time.Now(),
		}, nil
	}

	text, _ := client.Text()

	asset := &Asset{
		ID:         assetID,
		ProjectID:  projectID,
		FilePath:   destPath,
		OCRText:    text,
		UploadedAt: time.Now(),
	}

	// 4. Save metadata
	metaPath := filepath.Join(projectDir, assetID+".json")
	metaData, _ := json.MarshalIndent(asset, "", "  ")
	_ = os.WriteFile(metaPath, metaData, 0644)

	return asset, nil
}
