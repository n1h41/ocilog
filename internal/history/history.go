// Package history persists previously executed searches to disk.
package history

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// maxEntries caps how many searches are retained.
const maxEntries = 50

// Entry is a previously executed search: its query, pipeline, and time range.
type Entry struct {
	Query string `json:"query"`
	Pipe  string `json:"pipe,omitempty"`
	From  string `json:"from"`
	To    string `json:"to"`
}

// Store is a most-recent-first list of searches backed by a JSON file.
type Store struct {
	path    string
	entries []Entry
}

// Load reads the history from the user config directory. A missing or
// unreadable file yields an empty store rather than an error so the UI can
// always start.
func Load() *Store {
	s := &Store{entries: []Entry{}}

	dir, err := os.UserConfigDir()
	if err != nil {
		return s
	}
	s.path = filepath.Join(dir, "ocilog", "history.json")

	data, err := os.ReadFile(s.path)
	if err != nil {
		return s
	}
	if entries, ok := decode(data); ok {
		s.entries = entries
	}
	return s
}

// decode accepts both the current object format and the legacy string list.
func decode(data []byte) ([]Entry, bool) {
	var entries []Entry
	if err := json.Unmarshal(data, &entries); err == nil {
		return entries, true
	}
	var legacy []string
	if err := json.Unmarshal(data, &legacy); err == nil {
		entries = make([]Entry, 0, len(legacy))
		for _, q := range legacy {
			entries = append(entries, Entry{Query: q})
		}
		return entries, true
	}
	return nil, false
}

// Entries returns the stored searches, most recent first.
func (s *Store) Entries() []Entry { return s.entries }

// Add records entry at the front, dropping any earlier entry with the same
// query and pipeline, then persists the list. The time range is treated as
// metadata: re-running the same query and pipeline refreshes the existing
// entry rather than creating a duplicate.
func (s *Store) Add(entry Entry) error {
	entry.Query = strings.TrimSpace(entry.Query)
	entry.Pipe = strings.TrimSpace(entry.Pipe)
	entry.From = strings.TrimSpace(entry.From)
	entry.To = strings.TrimSpace(entry.To)
	if entry.Query == "" {
		return nil
	}

	entries := []Entry{entry}
	for _, e := range s.entries {
		if e.Query != entry.Query || e.Pipe != entry.Pipe {
			entries = append(entries, e)
		}
	}
	if len(entries) > maxEntries {
		entries = entries[:maxEntries]
	}
	s.entries = entries
	return s.save()
}

// Clear removes every stored search and persists the empty list.
func (s *Store) Clear() error {
	s.entries = []Entry{}
	return s.save()
}

func (s *Store) save() error {
	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s.entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o644)
}
