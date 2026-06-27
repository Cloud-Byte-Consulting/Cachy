package platform

import "testing"

func TestResolvePathsLinuxUsesXDGDirectories(t *testing.T) {
	t.Parallel()

	paths, err := ResolvePaths(Options{
		GOOS:    "linux",
		HomeDir: "/home/alex",
		Env: map[string]string{
			"XDG_CONFIG_HOME": "/cfg",
			"XDG_STATE_HOME":  "/state",
			"XDG_CACHE_HOME":  "/cache",
		},
	})
	if err != nil {
		t.Fatalf("ResolvePaths() error = %v", err)
	}

	assertPaths(t, paths, Paths{
		ConfigDir: "/cfg/cachy",
		StateDir:  "/state/cachy",
		CacheDir:  "/cache/cachy",
	})
}

func TestResolvePathsLinuxFallsBackToHome(t *testing.T) {
	t.Parallel()

	// Env must be non-nil to stay hermetic: with Env == nil the XDG lookups
	// fall through to the host's real os.Getenv (leaking XDG_CONFIG_HOME etc.).
	paths, err := ResolvePaths(Options{GOOS: "linux", HomeDir: "/home/alex", Env: map[string]string{}})
	if err != nil {
		t.Fatalf("ResolvePaths() error = %v", err)
	}

	assertPaths(t, paths, Paths{
		ConfigDir: "/home/alex/.config/cachy",
		StateDir:  "/home/alex/.local/state/cachy",
		CacheDir:  "/home/alex/.cache/cachy",
	})
}

func TestResolvePathsDarwinUsesLibraryDirectories(t *testing.T) {
	t.Parallel()

	paths, err := ResolvePaths(Options{GOOS: "darwin", HomeDir: "/Users/alex"})
	if err != nil {
		t.Fatalf("ResolvePaths() error = %v", err)
	}

	assertPaths(t, paths, Paths{
		ConfigDir: "/Users/alex/Library/Application Support/Cachy",
		StateDir:  "/Users/alex/Library/Application Support/Cachy",
		CacheDir:  "/Users/alex/Library/Caches/Cachy",
	})
}

func TestResolvePathsWindowsUsesAppData(t *testing.T) {
	t.Parallel()

	paths, err := ResolvePaths(Options{
		GOOS: "windows",
		Env: map[string]string{
			"APPDATA":      `C:\Users\alex\AppData\Roaming`,
			"LOCALAPPDATA": `C:\Users\alex\AppData\Local`,
		},
	})
	if err != nil {
		t.Fatalf("ResolvePaths() error = %v", err)
	}

	assertPaths(t, paths, Paths{
		ConfigDir: `C:\Users\alex\AppData\Roaming\Cachy`,
		StateDir:  `C:\Users\alex\AppData\Local\Cachy`,
		CacheDir:  `C:\Users\alex\AppData\Local\Cachy\cache`,
	})
}

func TestResolvePathsRejectsMissingHome(t *testing.T) {
	t.Parallel()

	if _, err := ResolvePaths(Options{GOOS: "linux", Env: map[string]string{}}); err == nil {
		t.Fatal("ResolvePaths() error = nil, want missing home error")
	}
}

func assertPaths(t *testing.T, got, want Paths) {
	t.Helper()

	if got != want {
		t.Fatalf("paths = %#v, want %#v", got, want)
	}
}
