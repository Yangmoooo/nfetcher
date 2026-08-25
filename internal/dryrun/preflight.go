package dryrun

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"nfetcher/internal/config"
)

type CheckStatus string

const (
	StatusOK   CheckStatus = "ok"
	StatusWarn CheckStatus = "warn"
	StatusFail CheckStatus = "fail"
)

type Check struct {
	Name   string
	Status CheckStatus
	Detail string
}

type Report struct {
	Checks []Check
}

func RunPreflight(cfg config.Config) Report {
	checks := []Check{
		checkTimezone(cfg),
		checkLibraryDir(cfg),
		checkProxyEnv(),
	}

	if check, ok := checkHostDockerInternal(); ok {
		checks = append(checks, check)
	}

	return Report{Checks: checks}
}

func (r Report) Failure() error {
	failures := make([]error, 0)
	for _, check := range r.Checks {
		if check.Status != StatusFail {
			continue
		}
		failures = append(failures, fmt.Errorf("%s: %s", check.Name, check.Detail))
	}

	return errors.Join(failures...)
}

func (r Report) WarningCount() int {
	count := 0
	for _, check := range r.Checks {
		if check.Status == StatusWarn {
			count++
		}
	}
	return count
}

func (r Report) FailureCount() int {
	count := 0
	for _, check := range r.Checks {
		if check.Status == StatusFail {
			count++
		}
	}
	return count
}

func checkTimezone(cfg config.Config) Check {
	if _, err := time.LoadLocation(cfg.Timezone); err != nil {
		return Check{
			Name:   "timezone",
			Status: StatusFail,
			Detail: err.Error(),
		}
	}

	return Check{
		Name:   "timezone",
		Status: StatusOK,
		Detail: cfg.Timezone,
	}
}

func checkLibraryDir(cfg config.Config) Check {
	libraryPath := cfg.LibraryPath()
	info, err := os.Stat(libraryPath)
	switch {
	case err == nil:
		if !info.IsDir() {
			return Check{
				Name:   "library_dir",
				Status: StatusFail,
				Detail: fmt.Sprintf("%s exists but is not a directory", libraryPath),
			}
		}

		if err := probeWritable(libraryPath); err != nil {
			return Check{
				Name:   "library_dir",
				Status: StatusFail,
				Detail: fmt.Sprintf("%s is not writable: %v", libraryPath, err),
			}
		}

		return Check{
			Name:   "library_dir",
			Status: StatusOK,
			Detail: fmt.Sprintf("%s exists and is writable", libraryPath),
		}
	case os.IsNotExist(err):
		parentDir := filepath.Dir(libraryPath)
		parentInfo, parentErr := os.Stat(parentDir)
		if parentErr != nil {
			return Check{
				Name:   "library_dir",
				Status: StatusFail,
				Detail: fmt.Sprintf("%s does not exist and parent %s is unavailable: %v", libraryPath, parentDir, parentErr),
			}
		}

		if !parentInfo.IsDir() {
			return Check{
				Name:   "library_dir",
				Status: StatusFail,
				Detail: fmt.Sprintf("parent %s is not a directory", parentDir),
			}
		}

		if err := probeWritable(parentDir); err != nil {
			return Check{
				Name:   "library_dir",
				Status: StatusFail,
				Detail: fmt.Sprintf("%s does not exist and parent %s is not writable: %v", libraryPath, parentDir, err),
			}
		}

		return Check{
			Name:   "library_dir",
			Status: StatusOK,
			Detail: fmt.Sprintf("%s does not exist yet, but parent %s is writable", libraryPath, parentDir),
		}
	default:
		return Check{
			Name:   "library_dir",
			Status: StatusFail,
			Detail: err.Error(),
		}
	}
}

func checkProxyEnv() Check {
	httpProxy := strings.TrimSpace(os.Getenv("HTTP_PROXY"))
	httpsProxy := strings.TrimSpace(os.Getenv("HTTPS_PROXY"))

	configured := make([]string, 0, 2)
	if httpProxy != "" {
		configured = append(configured, "HTTP_PROXY")
	}
	if httpsProxy != "" {
		configured = append(configured, "HTTPS_PROXY")
	}

	if len(configured) == 0 {
		return Check{
			Name:   "proxy_env",
			Status: StatusOK,
			Detail: "not configured",
		}
	}

	return Check{
		Name:   "proxy_env",
		Status: StatusOK,
		Detail: fmt.Sprintf("configured: %s", strings.Join(configured, ",")),
	}
}

func checkHostDockerInternal() (Check, bool) {
	for _, key := range []string{"HTTP_PROXY", "HTTPS_PROXY"} {
		rawValue := strings.TrimSpace(os.Getenv(key))
		if rawValue == "" {
			continue
		}

		parsed, err := url.Parse(rawValue)
		if err != nil {
			return Check{
				Name:   "host_docker_internal",
				Status: StatusFail,
				Detail: fmt.Sprintf("%s is not a valid URL: %v", key, err),
			}, true
		}

		if !strings.EqualFold(parsed.Hostname(), "host.docker.internal") {
			continue
		}

		addresses, err := net.LookupHost(parsed.Hostname())
		if err != nil {
			return Check{
				Name:   "host_docker_internal",
				Status: StatusFail,
				Detail: fmt.Sprintf("%s cannot resolve host.docker.internal: %v", key, err),
			}, true
		}

		return Check{
			Name:   "host_docker_internal",
			Status: StatusOK,
			Detail: fmt.Sprintf("%s resolved to %s", key, strings.Join(addresses, ",")),
		}, true
	}

	return Check{}, false
}

func probeWritable(dir string) error {
	probeFile, err := os.CreateTemp(dir, ".nfetcher-preflight-*")
	if err != nil {
		return err
	}
	probePath := probeFile.Name()
	if err := probeFile.Close(); err != nil {
		_ = os.Remove(probePath)
		return err
	}
	return os.Remove(probePath)
}
