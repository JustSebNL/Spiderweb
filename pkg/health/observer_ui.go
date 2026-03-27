package health

import (
	"net/http"
	"os"
	"path/filepath"
)

func (s *Server) observerUIHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	htmlPath, ok := resolveObserverUIPath()
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("observer UI not found; open ui/dashboard/observer.html from the repo checkout"))
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	http.ServeFile(w, r, htmlPath)
}

func resolveObserverUIPath() (string, bool) {
	if v := os.Getenv("SPIDERWEB_OBSERVER_UI_HTML"); v != "" {
		if fileExists(v) {
			return v, true
		}
	}

	cwd, _ := os.Getwd()
	if p, ok := findRepoRelativeFile(cwd, filepath.Join("ui", "dashboard", "observer.html"), 6); ok {
		return p, true
	}

	exe, err := os.Executable()
	if err == nil {
		if p, ok := findRepoRelativeFile(filepath.Dir(exe), filepath.Join("ui", "dashboard", "observer.html"), 6); ok {
			return p, true
		}
	}

	return "", false
}

func findRepoRelativeFile(startDir string, relativePath string, maxDepth int) (string, bool) {
	dir := startDir
	for i := 0; i <= maxDepth; i++ {
		candidate := filepath.Join(dir, relativePath)
		if fileExists(candidate) {
			return candidate, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", false
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}
