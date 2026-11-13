package privilege

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/0xamirreza/portpilot/internal/core/firewall"
)

const schedulePath = "/var/lib/portpilot/scheduled.json"

// ScheduleFile is the root-owned list of temporary rules. User config is not consulted.
type ScheduleFile struct {
	Items []firewall.Schedule `json:"items"`
}

func loadSchedule() (ScheduleFile, error) {
	raw, err := os.ReadFile(schedulePath)
	if err != nil {
		if os.IsNotExist(err) {
			return ScheduleFile{}, nil
		}
		return ScheduleFile{}, err
	}
	var file ScheduleFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return ScheduleFile{}, err
	}
	return file, nil
}

func saveSchedule(file ScheduleFile) error {
	if err := os.MkdirAll(filepath.Dir(schedulePath), 0o700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}
	tmp := schedulePath + ".tmp"
	if err := os.WriteFile(tmp, append(raw, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, schedulePath)
}

func remember(item firewall.Schedule) error {
	file, err := loadSchedule()
	if err != nil {
		return err
	}
	file.Items = append(file.Items, item)
	return saveSchedule(file)
}

func forget(id string) error {
	file, err := loadSchedule()
	if err != nil {
		return err
	}
	next := file.Items[:0]
	for _, item := range file.Items {
		if item.ID != id {
			next = append(next, item)
		}
	}
	file.Items = next
	return saveSchedule(file)
}

func due(now time.Time) ([]firewall.Schedule, error) {
	file, err := loadSchedule()
	if err != nil {
		return nil, err
	}
	var out []firewall.Schedule
	for _, item := range file.Items {
		if !item.Expires.After(now) {
			out = append(out, item)
		}
	}
	return out, nil
}
