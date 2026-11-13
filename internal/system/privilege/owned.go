package privilege

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/0xamirreza/portpilot/internal/core/firewall"
)

const ownedPath = "/var/lib/portpilot/owned.json"

type ownedFile struct {
	Rules []ownedRule `json:"rules"`
}

type ownedRule struct {
	ID      string           `json:"id"`
	Request firewall.Request `json:"request"`
}

func loadOwned() (ownedFile, error) {
	raw, err := os.ReadFile(ownedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return ownedFile{}, nil
		}
		return ownedFile{}, err
	}
	var file ownedFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return ownedFile{}, err
	}
	return file, nil
}

func saveOwned(file ownedFile) error {
	if err := os.MkdirAll(filepath.Dir(ownedPath), 0o700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}
	tmp := ownedPath + ".tmp"
	if err := os.WriteFile(tmp, append(raw, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, ownedPath)
}

func rememberOwned(id string, req firewall.Request) error {
	file, err := loadOwned()
	if err != nil {
		return err
	}
	file.Rules = append(file.Rules, ownedRule{ID: id, Request: req})
	return saveOwned(file)
}

func forgetOwned(id string) error {
	file, err := loadOwned()
	if err != nil {
		return err
	}
	next := file.Rules[:0]
	for _, rule := range file.Rules {
		if rule.ID != id {
			next = append(next, rule)
		}
	}
	file.Rules = next
	return saveOwned(file)
}
