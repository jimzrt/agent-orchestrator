//go:build linux

package browserchrome

import "path/filepath"

func defaultBrowsers(home string) []Browser {
	return browsersUnder(filepath.Join(home, ".config"),
		[2]string{"Google Chrome", "google-chrome"},
		[2]string{"Google Chrome Beta", "google-chrome-beta"},
		[2]string{"Google Chrome Dev", "google-chrome-unstable"},
		[2]string{"Google Chrome for Testing", "google-chrome-for-testing"},
		[2]string{"Chromium", "chromium"},
		[2]string{"Brave", "BraveSoftware/Brave-Browser"},
		[2]string{"Microsoft Edge", "microsoft-edge"},
		[2]string{"Vivaldi", "vivaldi"},
	)
}

func platformRegistry() hostRegistry { return nil }
