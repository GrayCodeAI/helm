// Package voice provides voice note recording and transcription.
package voice

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Recorder handles voice recording using system tools.
type Recorder struct {
	outputDir string
}

// NewRecorder creates a new voice recorder.
func NewRecorder(outputDir string) *Recorder {
	os.MkdirAll(outputDir, 0755)
	return &Recorder{outputDir: outputDir}
}

// Record starts recording for the given duration.
func (r *Recorder) Record(duration time.Duration) (string, error) {
	filename := fmt.Sprintf("voice_%d.wav", time.Now().Unix())
	path := filepath.Join(r.outputDir, filename)

	// Try arecord (Linux)
	cmd := exec.Command("arecord", "-d", fmt.Sprintf("%d", int(duration.Seconds())), "-f", "cd", path)
	if err := cmd.Run(); err == nil {
		return path, nil
	}

	// Try sox/rec (macOS/Linux)
	cmd = exec.Command("rec", "-d", path, "trim", "0", fmt.Sprintf("%d", int(duration.Seconds())))
	if err := cmd.Run(); err == nil {
		return path, nil
	}

	// Try ffmpeg
	cmd = exec.Command("ffmpeg", "-f", "avfoundation", "-i", ":0", "-t", fmt.Sprintf("%d", int(duration.Seconds())), path)
	if err := cmd.Run(); err == nil {
		return path, nil
	}

	return "", fmt.Errorf("no recording tool available (tried arecord, rec, ffmpeg)")
}

// Transcriber transcribes audio to text using Whisper API or local model.
type Transcriber struct {
	apiKey   string
	apiURL   string
	useLocal bool
}

// NewTranscriber creates a new transcriber.
func NewTranscriber(apiKey string, useLocal bool) *Transcriber {
	url := "https://api.openai.com/v1/audio/transcriptions"
	return &Transcriber{apiKey: apiKey, apiURL: url, useLocal: useLocal}
}

// Transcribe transcribes an audio file to text.
func (t *Transcriber) Transcribe(audioPath string) (string, error) {
	if t.useLocal {
		return t.transcribeLocal(audioPath)
	}
	return t.transcribeAPI(audioPath)
}

func (t *Transcriber) transcribeLocal(audioPath string) (string, error) {
	// Check if whisper-cli is available (local Whisper installation)
	cmd := exec.Command("whisper", audioPath, "--model", "base", "--output_format", "txt")
	output, err := cmd.CombinedOutput()
	if err == nil {
		// Read the output file
		txtPath := strings.TrimSuffix(audioPath, filepath.Ext(audioPath)) + ".txt"
		content, readErr := os.ReadFile(txtPath)
		if readErr == nil {
			return strings.TrimSpace(string(content)), nil
		}
	}

	// Try whisper.cpp
	cmd = exec.Command("./whisper", audioPath, "-m", "models/ggml-base.bin", "-otxt")
	output, err = cmd.CombinedOutput()
	if err == nil {
		txtPath := strings.TrimSuffix(audioPath, filepath.Ext(audioPath)) + ".txt"
		content, readErr := os.ReadFile(txtPath)
		if readErr == nil {
			return strings.TrimSpace(string(content)), nil
		}
	}

	return "", fmt.Errorf("local transcription failed: %s", string(output))
}

func (t *Transcriber) transcribeAPI(audioPath string) (string, error) {
	if t.apiKey == "" {
		return "", fmt.Errorf("no API key provided for transcription")
	}

	// Open the audio file
	file, err := os.Open(audioPath)
	if err != nil {
		return "", fmt.Errorf("open audio file: %w", err)
	}
	defer file.Close()

	// Create multipart form
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Add file field
	part, err := writer.CreateFormFile("file", filepath.Base(audioPath))
	if err != nil {
		return "", fmt.Errorf("create form file: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return "", fmt.Errorf("copy file: %w", err)
	}

	// Add model field
	if err := writer.WriteField("model", "whisper-1"); err != nil {
		return "", fmt.Errorf("write model field: %w", err)
	}

	// Add response_format field
	if err := writer.WriteField("response_format", "json"); err != nil {
		return "", fmt.Errorf("write format field: %w", err)
	}

	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("close writer: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, t.apiURL, &body)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+t.apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Send request
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse response
	var result struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	return result.Text, nil
}

// VoiceNote represents a recorded voice note with transcription.
type VoiceNote struct {
	ID            string
	AudioPath     string
	Transcription string
	CreatedAt     time.Time
	Duration      time.Duration
	SessionID     string
}

// Manager manages voice notes lifecycle.
type Manager struct {
	recorder    *Recorder
	transcriber *Transcriber
	notes       []VoiceNote
}

// NewManager creates a voice note manager.
func NewManager(outputDir string, apiKey string, useLocal bool) *Manager {
	return &Manager{
		recorder:    NewRecorder(outputDir),
		transcriber: NewTranscriber(apiKey, useLocal),
		notes:       []VoiceNote{},
	}
}

// RecordAndTranscribe records audio and transcribes it.
func (m *Manager) RecordAndTranscribe(duration time.Duration) (*VoiceNote, error) {
	path, err := m.recorder.Record(duration)
	if err != nil {
		return nil, err
	}

	note := &VoiceNote{
		ID:        fmt.Sprintf("voice_%d", time.Now().Unix()),
		AudioPath: path,
		CreatedAt: time.Now(),
		Duration:  duration,
	}

	text, err := m.transcriber.Transcribe(path)
	if err != nil {
		note.Transcription = fmt.Sprintf("[transcription failed: %v]", err)
	} else {
		note.Transcription = text
	}

	m.notes = append(m.notes, *note)
	return note, nil
}

// TranscribeFile transcribes an existing audio file.
func (m *Manager) TranscribeFile(audioPath string) (string, error) {
	return m.transcriber.Transcribe(audioPath)
}

// GetNotes returns all voice notes.
func (m *Manager) GetNotes() []VoiceNote {
	return m.notes
}

// GetNote returns a specific note by ID.
func (m *Manager) GetNote(id string) (*VoiceNote, error) {
	for _, note := range m.notes {
		if note.ID == id {
			return &note, nil
		}
	}
	return nil, fmt.Errorf("note %s not found", id)
}

// Search searches transcriptions for query.
func (m *Manager) Search(query string) []VoiceNote {
	var results []VoiceNote
	query = strings.ToLower(query)
	for _, note := range m.notes {
		if strings.Contains(strings.ToLower(note.Transcription), query) {
			results = append(results, note)
		}
	}
	return results
}

// AttachToSession attaches a voice note to a session.
func (m *Manager) AttachToSession(noteID, sessionID string) error {
	for i, note := range m.notes {
		if note.ID == noteID {
			m.notes[i].SessionID = sessionID
			return nil
		}
	}
	return fmt.Errorf("note %s not found", noteID)
}

// DeleteNote removes a voice note and its audio file.
func (m *Manager) DeleteNote(id string) error {
	for i, note := range m.notes {
		if note.ID == id {
			os.Remove(note.AudioPath)
			m.notes = append(m.notes[:i], m.notes[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("note %s not found", id)
}
