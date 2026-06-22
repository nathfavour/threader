package media

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/otiai10/gosseract/v2"
	_ "modernc.org/sqlite"
)

type DB struct {
	db *sql.DB
}

func OpenDB(projectDir string) (*DB, error) {
	_ = os.MkdirAll(projectDir, 0755)
	dbPath := filepath.Join(projectDir, "media.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	schema := `
	CREATE TABLE IF NOT EXISTS assets (
		id TEXT PRIMARY KEY,
		file_path TEXT UNIQUE,
		ocr_text TEXT,
		ai_summary TEXT,
		uploaded_at TEXT,
		posted INTEGER,
		thread_id TEXT,
		posted_at TEXT
	);`
	_, err = db.Exec(schema)
	if err != nil {
		db.Close()
		return nil, err
	}

	return &DB{db: db}, nil
}

func (d *DB) Close() error {
	return d.db.Close()
}

func (d *DB) AddAsset(a *Asset) error {
	_, err := d.db.Exec(`
		INSERT INTO assets (id, file_path, ocr_text, ai_summary, uploaded_at, posted, thread_id, posted_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(file_path) DO UPDATE SET
			ocr_text=excluded.ocr_text,
			ai_summary=excluded.ai_summary,
			posted=excluded.posted,
			thread_id=excluded.thread_id,
			posted_at=excluded.posted_at
	`, a.ID, a.FilePath, a.OCRText, a.AISummary, a.UploadedAt.Format(time.RFC3339), boolToInt(a.Posted), a.ThreadID, formatTime(a.PostedAt))
	return err
}

func (d *DB) GetUnpostedAssets() ([]*Asset, error) {
	rows, err := d.db.Query(`SELECT id, file_path, ocr_text, ai_summary, uploaded_at, posted, thread_id, posted_at FROM assets WHERE posted = 0 ORDER BY uploaded_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assets []*Asset
	for rows.Next() {
		var a Asset
		var uploadedStr, postedAtStr sql.NullString
		var postedInt int
		err := rows.Scan(&a.ID, &a.FilePath, &a.OCRText, &a.AISummary, &uploadedStr, &postedInt, &a.ThreadID, &postedAtStr)
		if err != nil {
			return nil, err
		}
		if uploadedStr.Valid {
			a.UploadedAt, _ = time.Parse(time.RFC3339, uploadedStr.String)
		}
		a.Posted = postedInt != 0
		if postedAtStr.Valid && postedAtStr.String != "" {
			a.PostedAt, _ = time.Parse(time.RFC3339, postedAtStr.String)
		}
		assets = append(assets, &a)
	}
	return assets, nil
}

func (d *DB) MarkPosted(id string, threadID string) error {
	_, err := d.db.Exec(`UPDATE assets SET posted = 1, thread_id = ?, posted_at = ? WHERE id = ?`, threadID, time.Now().Format(time.RFC3339), id)
	return err
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

func ScanAndIndex(projectID string, projectMediaDir string, projectDir string) error {
	_ = os.MkdirAll(projectMediaDir, 0755)

	db, err := OpenDB(projectDir)
	if err != nil {
		return err
	}
	defer db.Close()

	files, err := os.ReadDir(projectMediaDir)
	if err != nil {
		return err
	}

	for _, f := range files {
		if f.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(f.Name()))
		if ext == ".png" || ext == ".jpg" || ext == ".jpeg" {
			filePath := filepath.Join(projectMediaDir, f.Name())
			
			// Check if already indexed
			var exists bool
			err := db.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM assets WHERE file_path = ?)`, filePath).Scan(&exists)
			if err != nil || exists {
				continue
			}

			// Perform OCR
			fmt.Printf("🧵 SQLite Indexer: OCR indexing new media file %s...\n", f.Name())
			client := gosseract.NewClient()
			ocrText := ""
			if client != nil {
				_ = client.SetImage(filePath)
				ocrText, _ = client.Text()
				client.Close()
			}

			assetID := uuid.New().String()
			asset := &Asset{
				ID:         assetID,
				ProjectID:  projectID,
				FilePath:   filePath,
				OCRText:    ocrText,
				UploadedAt: time.Now(),
				Posted:     false,
			}
			_ = db.AddAsset(asset)
		}
	}
	return nil
}
