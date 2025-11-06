package models

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
)

type SnapshotResource struct {
	Filename string `json:"filename"`
	Mime     string `json:"mime"`
	Hash     string `json:"hash"`
	Data     []byte `json:"-"`
}

var dataURLPattern = regexp.MustCompile(`data:([^;]+)(;[^,]+)*;base64,([A-Za-z0-9+/=]+)`) //nolint:lll

// CollectSnapshotResources extracts unique embedded data URL resources from the snapshot.
func CollectSnapshotResources(snapshot *Snapshot) []SnapshotResource {
	if snapshot == nil {
		return nil
	}

	seen := make(map[string]struct{})
	var resources []SnapshotResource

	addResource := func(mimeType, payload string) {
		if mimeType == "" || payload == "" {
			return
		}
		mimeType = strings.ToLower(strings.TrimSpace(mimeType))
		decoded, err := base64.StdEncoding.DecodeString(payload)
		if err != nil {
			return
		}
		hash := sha256.Sum256(decoded)
		hashKey := hex.EncodeToString(hash[:])
		if _, ok := seen[hashKey]; ok {
			return
		}
		seen[hashKey] = struct{}{}
		filename := hashKey
		if ext := extensionForMime(mimeType); ext != "" {
			filename = filename + ext
		}
		resources = append(resources, SnapshotResource{
			Filename: filename,
			Mime:     mimeType,
			Hash:     hashKey,
			Data:     decoded,
		})
	}

	collect := func(content string) {
		if !strings.Contains(content, "data:") {
			return
		}
		matches := dataURLPattern.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			if len(match) < 4 {
				continue
			}
			addResource(match[1], match[3])
		}
	}

	for _, workspace := range snapshot.Workspaces {
		if workspace == nil {
			continue
		}
		collect(workspace.Document)
		for _, row := range workspace.Rows {
			for _, value := range row.Cells {
				collect(value)
			}
		}
	}

	return resources
}

func extensionForMime(mimeType string) string {
	switch mimeType {
	case "image/png":
		return ".png"
	case "image/jpeg":
		return ".jpg"
	case "image/jpg":
		return ".jpg"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "image/svg+xml":
		return ".svg"
	case "audio/mpeg":
		return ".mp3"
	case "audio/wav":
		return ".wav"
	case "audio/ogg":
		return ".ogg"
	case "video/mp4":
		return ".mp4"
	case "video/webm":
		return ".webm"
	case "video/ogg":
		return ".ogv"
	default:
		if strings.HasPrefix(mimeType, "image/") {
			return ".img"
		}
		if strings.HasPrefix(mimeType, "audio/") {
			return ".aud"
		}
		if strings.HasPrefix(mimeType, "video/") {
			return ".vid"
		}
		return ""
	}
}

// EmbedSnapshotResources replaces resource file references with data URLs sourced from the provided assets.
func EmbedSnapshotResources(snapshot *Snapshot, assets []SnapshotResource) {
	if snapshot == nil || len(assets) == 0 {
		return
	}
	replacements := make([]string, 0, len(assets)*2)
	for _, asset := range assets {
		if len(asset.Data) == 0 {
			continue
		}
		mimeType := asset.Mime
		if strings.TrimSpace(mimeType) == "" {
			mimeType = "application/octet-stream"
		}
		dataURL := fmt.Sprintf("data:%s;base64,%s", mimeType, base64.StdEncoding.EncodeToString(asset.Data))
		replacements = append(replacements, fmt.Sprintf("resources/%s", asset.Filename), dataURL)
	}
	if len(replacements) == 0 {
		return
	}
	replacer := strings.NewReplacer(replacements...)
	for _, workspace := range snapshot.Workspaces {
		if workspace == nil {
			continue
		}
		if strings.Contains(workspace.Document, "resources/") {
			workspace.Document = replacer.Replace(workspace.Document)
		}
		for rowIndex := range workspace.Rows {
			row := &workspace.Rows[rowIndex]
			if row.Cells == nil {
				continue
			}
			for key, value := range row.Cells {
				if strings.Contains(value, "resources/") {
					row.Cells[key] = replacer.Replace(value)
				}
			}
		}
	}
}
