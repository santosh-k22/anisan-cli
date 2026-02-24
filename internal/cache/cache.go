// Package cache provides localized filesystem-based caching for transient media metadata and provider results.
package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const TTL = 7 * 24 * time.Hour

func getDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.TempDir()
	}
	dir := filepath.Join(home, ".cache", "anisan")
	_ = os.MkdirAll(dir, 0755)
	return dir
}

// GenerateKey generates a deterministic SHA-256 hash from a query and provider pair for use as a cache identifier.
func GenerateKey(query, provider string) string {
	sanitized := strings.ToLower(strings.ReplaceAll(query, " ", "")) + provider
	hash := sha256.Sum256([]byte(sanitized))
	return hex.EncodeToString(hash[:])
}

// Read attempts to retrieve and deserialize a cached object if it exists and has not exceeded its TTL.
func Read(key string, target interface{}) bool {
	path := filepath.Join(getDir(), key)

	info, err := os.Stat(path)
	if err != nil || time.Since(info.ModTime()) > TTL {
		return false
	}

	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	// Decode directly into the target interface.
	decoder := json.NewDecoder(f)
	if err := decoder.Decode(target); err != nil {
		return false
	}
	return true
}

// Write persists a serializable object to the cache using an atomic file swap to ensure data integrity.
func Write(key string, data interface{}) error {
	path := filepath.Join(getDir(), key)
	tmpPath := path + ".tmp"

	f, err := os.Create(tmpPath)
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(f)
	if err := encoder.Encode(data); err != nil {
		f.Close()
		return err
	}
	f.Close()

	return os.Rename(tmpPath, path)
}

// CollectGarbage initializes an asynchronous background task to prune expired cache entries from the filesystem.
func CollectGarbage() {
	go func() {
		dir := getDir()
		_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if info, err := d.Info(); err == nil && time.Since(info.ModTime()) > TTL {
				_ = os.Remove(path)
			}
			return nil
		})
	}()
}
