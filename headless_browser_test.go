package headless_browser

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/launcher/flags"
	"github.com/go-rod/rod/lib/proto"
)

// chromeBin returns the browser binary to test against. Set HEADLESS_BROWSER_BIN
// to point at a local Chrome/Chromium; otherwise the test is skipped so CI
// without a browser stays green.
func chromeBin(t *testing.T) string {
	t.Helper()

	if bin := os.Getenv("HEADLESS_BROWSER_BIN"); bin != "" {
		return bin
	}
	if bin, ok := launcher.LookPath(); ok {
		return bin
	}
	t.Skip("no browser binary found; set HEADLESS_BROWSER_BIN")
	return ""
}

// TestUserDataDirSurvivesClose is the regression test for the fork: a profile
// directory handed in via WithUserDataDir must still be there, with its
// contents, after Close(). Upstream's Close() calls launcher.Cleanup(), which
// does an unconditional os.RemoveAll of the user data dir.
func TestUserDataDirSurvivesClose(t *testing.T) {
	bin := chromeBin(t)

	// Not t.TempDir(): we want to see the directory with our own eyes after
	// Close, and the cleanup of a deleted dir would mask nothing anyway.
	dir, err := os.MkdirTemp("", "hb-profile-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	var pages int

	b := New(
		WithHeadless(true),
		WithChromeBinPath(bin),
		WithUserDataDir(dir),
		WithLauncherHook(func(l *launcher.Launcher) {
			if got := l.Get(flags.UserDataDir); got != dir {
				t.Errorf("launcher user-data-dir = %q, want %q", got, dir)
			}
			l.Delete("enable-automation")
		}),
		WithPageHook(func(p *rod.Page) error {
			pages++
			return proto.EmulationSetDeviceMetricsOverride{
				Width: 1280, Height: 800, DeviceScaleFactor: 1,
			}.Call(p)
		}),
	)

	if b.Rod() == nil {
		t.Fatal("Rod() returned nil")
	}

	page := b.NewPage()
	page.Timeout(30 * time.Second).MustNavigate("about:blank").MustWaitLoad()
	page.MustClose()

	if pages != 1 {
		t.Errorf("page hook ran %d times, want 1", pages)
	}

	b.Close()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("user data dir gone after Close: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("user data dir is empty after Close, expected a chrome profile")
	}
	if _, err := os.Stat(filepath.Join(dir, "Default")); err != nil {
		t.Errorf("no Default profile in user data dir: %v", err)
	}
}

// TestTempUserDataDirRemovedOnClose pins the unchanged behaviour: without
// WithUserDataDir, rod's temp profile is still cleaned up.
func TestTempUserDataDirRemovedOnClose(t *testing.T) {
	bin := chromeBin(t)

	var dir string
	b := New(
		WithHeadless(true),
		WithChromeBinPath(bin),
		WithLauncherHook(func(l *launcher.Launcher) {
			dir = l.Get(flags.UserDataDir)
		}),
	)
	if dir == "" {
		t.Fatal("launcher had no user-data-dir")
	}

	page := b.NewPage()
	page.Timeout(30 * time.Second).MustNavigate("about:blank").MustWaitLoad()
	page.MustClose()
	b.Close()

	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("temp user data dir %q still present after Close (err=%v)", dir, err)
	}
}
