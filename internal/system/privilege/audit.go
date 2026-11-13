package privilege

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func audit(op string, ok bool, detail string) {
	dir := "/var/lib/portpilot"
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return
	}
	f, err := os.OpenFile(filepath.Join(dir, "audit.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	result := "SUCCESS"
	if !ok {
		result = "FAILED"
	}
	fmt.Fprintf(f, "%s %s %s %s\n", time.Now().UTC().Format(time.RFC3339), op, result, detail)
}
