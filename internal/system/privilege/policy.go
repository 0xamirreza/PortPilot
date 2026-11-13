package privilege

import (
	"fmt"
	"os"
	"path/filepath"
)

// InstallPolicy writes the PortPilot polkit action for this executable.
// auth_admin_keep lets one successful prompt cover later calls in the same session.
func InstallPolicy(exe string) error {
	if appimage := os.Getenv("APPIMAGE"); appimage != "" {
		exe = appimage
	}
	if exe == "" || !filepath.IsAbs(exe) {
		return fmt.Errorf("policy needs the absolute path of portpilot")
	}
	const dir = "/usr/share/polkit-1/actions"
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	body := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE policyconfig PUBLIC "-//freedesktop//DTD PolicyKit Policy Configuration 1.0//EN" "http://www.freedesktop.org/standards/PolicyKit/1/policyconfig.dtd">
<policyconfig>
  <action id="dev.portpilot.helper">
    <description>Run the PortPilot privileged helper</description>
    <message>Authentication is required to inspect or change firewall rules, services, or process signals.</message>
    <defaults>
      <allow_any>auth_admin</allow_any>
      <allow_inactive>auth_admin</allow_inactive>
      <allow_active>auth_admin_keep</allow_active>
    </defaults>
    <annotate key="org.freedesktop.policykit.exec.path">%s</annotate>
    <annotate key="org.freedesktop.policykit.exec.allow_gui">true</annotate>
  </action>
</policyconfig>
`, exe)
	path := filepath.Join(dir, "dev.portpilot.policy")
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(body), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
