package platform

import (
	"errors"
	"os"
	"runtime"
	"strings"
)

type Options struct {
	GOOS    string
	HomeDir string
	Env     map[string]string
}

type Paths struct {
	ConfigDir string
	StateDir  string
	CacheDir  string
}

func ResolvePaths(options Options) (Paths, error) {
	goos := options.GOOS
	if goos == "" {
		goos = runtime.GOOS
	}

	switch goos {
	case "windows":
		return resolveWindowsPaths(options)
	case "darwin":
		return resolveDarwinPaths(options)
	default:
		return resolveUnixPaths(options)
	}
}

func resolveWindowsPaths(options Options) (Paths, error) {
	roaming := env(options, "APPDATA")
	if roaming == "" {
		home, err := homeDir(options)
		if err != nil {
			return Paths{}, err
		}
		roaming = joinPath("windows", home, "AppData", "Roaming")
	}

	local := env(options, "LOCALAPPDATA")
	if local == "" {
		home, err := homeDir(options)
		if err != nil {
			return Paths{}, err
		}
		local = joinPath("windows", home, "AppData", "Local")
	}

	return Paths{
		ConfigDir: joinPath("windows", roaming, "Cachy"),
		StateDir:  joinPath("windows", local, "Cachy"),
		CacheDir:  joinPath("windows", local, "Cachy", "cache"),
	}, nil
}

func resolveDarwinPaths(options Options) (Paths, error) {
	home, err := homeDir(options)
	if err != nil {
		return Paths{}, err
	}

	appSupport := joinPath("darwin", home, "Library", "Application Support", "Cachy")
	return Paths{
		ConfigDir: appSupport,
		StateDir:  appSupport,
		CacheDir:  joinPath("darwin", home, "Library", "Caches", "Cachy"),
	}, nil
}

func resolveUnixPaths(options Options) (Paths, error) {
	home, err := homeDir(options)
	if err != nil {
		return Paths{}, err
	}

	configRoot := env(options, "XDG_CONFIG_HOME")
	if configRoot == "" {
		configRoot = joinPath("linux", home, ".config")
	}

	stateRoot := env(options, "XDG_STATE_HOME")
	if stateRoot == "" {
		stateRoot = joinPath("linux", home, ".local", "state")
	}

	cacheRoot := env(options, "XDG_CACHE_HOME")
	if cacheRoot == "" {
		cacheRoot = joinPath("linux", home, ".cache")
	}

	return Paths{
		ConfigDir: joinPath("linux", configRoot, "cachy"),
		StateDir:  joinPath("linux", stateRoot, "cachy"),
		CacheDir:  joinPath("linux", cacheRoot, "cachy"),
	}, nil
}

func homeDir(options Options) (string, error) {
	if options.HomeDir != "" {
		return options.HomeDir, nil
	}
	if home := env(options, "HOME"); home != "" {
		return home, nil
	}
	if home := env(options, "USERPROFILE"); home != "" {
		return home, nil
	}
	if options.Env == nil {
		home, err := os.UserHomeDir()
		if err == nil && home != "" {
			return home, nil
		}
	}
	return "", errors.New("home directory is required to resolve Cachy paths")
}

func env(options Options, key string) string {
	if options.Env != nil {
		return options.Env[key]
	}
	return os.Getenv(key)
}

func joinPath(goos string, parts ...string) string {
	separator := "/"
	if goos == "windows" {
		separator = `\`
	}

	cleaned := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.Trim(part, `/\`)
		if part == "" {
			continue
		}
		cleaned = append(cleaned, part)
	}
	if len(cleaned) == 0 {
		return ""
	}

	prefix := ""
	first := parts[0]
	if strings.HasPrefix(first, "/") && goos != "windows" {
		prefix = "/"
	}

	return prefix + strings.Join(cleaned, separator)
}
