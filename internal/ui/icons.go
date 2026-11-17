package ui

import (
	"path/filepath"
	"strings"
)

// Nerd Font glyphs (https://www.nerdfonts.com/cheat-sheet).
const (
	iconFolder  = ""
	iconLink    = ""
	iconFile    = ""
	iconText    = ""
	iconImage   = ""
	iconArchive = ""
	iconPDF     = ""
	iconVideo   = ""
	iconAudio   = ""
	iconShell   = ""
	iconConfig  = ""
	iconLock    = ""
	iconGit     = ""
	iconDocker  = ""
)

var iconsByExt = map[string]string{
	".go": "", ".mod": "", ".sum": "",
	".js": "", ".mjs": "", ".cjs": "", ".jsx": "",
	".ts": "", ".tsx": "",
	".py": "", ".rs": "", ".java": "", ".rb": "",
	".c": "", ".h": "", ".cpp": "", ".hpp": "",
	".php": "", ".lua": "", ".vim": "",
	".html": "", ".css": "", ".scss": "",
	".md": "", ".json": "",
	".yml": iconConfig, ".yaml": iconConfig, ".toml": iconConfig, ".ini": iconConfig, ".conf": iconConfig,
	".sh": iconShell, ".bash": iconShell, ".zsh": iconShell, ".fish": iconShell,
	".txt": iconText, ".log": iconText, ".csv": iconText,
	".png": iconImage, ".jpg": iconImage, ".jpeg": iconImage, ".gif": iconImage, ".svg": iconImage, ".webp": iconImage, ".ico": iconImage,
	".zip": iconArchive, ".tar": iconArchive, ".gz": iconArchive, ".tgz": iconArchive, ".xz": iconArchive, ".7z": iconArchive, ".rar": iconArchive, ".bz2": iconArchive,
	".pdf": iconPDF,
	".mp4": iconVideo, ".mkv": iconVideo, ".mov": iconVideo, ".webm": iconVideo, ".avi": iconVideo,
	".mp3": iconAudio, ".wav": iconAudio, ".flac": iconAudio, ".ogg": iconAudio,
	".lock": iconLock,
}

var iconsByName = map[string]string{
	".gitignore": iconGit, ".gitattributes": iconGit, ".gitmodules": iconGit,
	"Dockerfile": iconDocker, "docker-compose.yml": iconDocker, "compose.yml": iconDocker,
	"Makefile": iconShell,
}

// iconFor returns the Nerd Font glyph for an entry.
func iconFor(name string, isDir, isLink bool) string {
	switch {
	case isLink:
		return iconLink
	case isDir:
		return iconFolder
	}
	if icon, ok := iconsByName[name]; ok {
		return icon
	}
	if icon, ok := iconsByExt[strings.ToLower(filepath.Ext(name))]; ok {
		return icon
	}
	return iconFile
}
