//go:build linux

package browserchrome

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultBrowsersCoverChromiumFamilyOnLinux(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	want := map[string]string{
		"Google Chrome":  ".config/google-chrome",
		"Chromium":       ".config/chromium",
		"Brave":          ".config/BraveSoftware/Brave-Browser",
		"Microsoft Edge": ".config/microsoft-edge",
		"Vivaldi":        ".config/vivaldi",
	}
	got := map[string]string{}
	for _, browser := range defaultBrowsers(home) {
		got[browser.Name] = browser.UserDataDir
	}
	for name, rel := range want {
		if got[name] != filepath.Join(home, filepath.FromSlash(rel)) {
			t.Fatalf("%s user data dir = %q, want %q", name, got[name], filepath.Join(home, rel))
		}
	}
}

func TestSetupRegistersChromiumFamilyBrowsersInUse(t *testing.T) {
	installer, _, root := testInstaller(t)
	home := filepath.Join(root, "home")
	for _, dir := range []string{".config/chromium", ".config/BraveSoftware/Brave-Browser", ".config/vivaldi"} {
		if err := os.MkdirAll(filepath.Join(home, filepath.FromSlash(dir)), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	installer.Browsers = defaultBrowsers(home)

	status, err := installer.Setup()
	if err != nil {
		t.Fatal(err)
	}
	registered := map[string]bool{}
	for _, channel := range status.Channels {
		registered[channel.Name] = channel.Registered
	}
	for name, want := range map[string]bool{"Chromium": true, "Brave": true, "Vivaldi": true, "Microsoft Edge": false, "Google Chrome": false} {
		if registered[name] != want {
			t.Fatalf("%s registered = %v, want %v (%#v)", name, registered[name], want, status.Channels)
		}
	}
	for _, rel := range []string{".config/chromium", ".config/BraveSoftware/Brave-Browser", ".config/vivaldi"} {
		if _, err := os.Stat(filepath.Join(home, filepath.FromSlash(rel), "NativeMessagingHosts", HostName+".json")); err != nil {
			t.Fatalf("%s manifest missing: %v", rel, err)
		}
	}
	if _, err := os.Stat(filepath.Join(home, ".config", "microsoft-edge")); !os.IsNotExist(err) {
		t.Fatal("setup created an unused browser's directory")
	}
}
