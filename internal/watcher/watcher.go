// Package watcher provides directory watching with fsnotify, debouncing, and add/remove roots.
package watcher

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

const defaultDebounce = 400 * time.Millisecond

// Watcher watches directories and invokes callbacks on file changes.
type Watcher struct {
	roots      []string
	extensions  []string
	recursive   bool
	onIndex     func(path string)
	onRemove    func(path string)
	debounce    time.Duration
	watcher     *fsnotify.Watcher
	mu          sync.Mutex
	debounceMap map[string]*time.Timer
	rootPaths   map[string][]string // root -> list of watched paths (dirs we added)
	done        chan struct{}
	started     bool
	stopOnce    sync.Once
}

// NewWatcher creates a watcher. onIndex and onRemove are called for file index and remove events.
// roots are initial directory paths to watch; extensions filter which files (empty = all).
func NewWatcher(roots []string, extensions []string, recursive bool, onIndex, onRemove func(path string)) *Watcher {
	return &Watcher{
		roots:       roots,
		extensions:  extensions,
		recursive:   recursive,
		onIndex:     onIndex,
		onRemove:    onRemove,
		debounce:    defaultDebounce,
		debounceMap: make(map[string]*time.Timer),
		rootPaths:   make(map[string][]string),
		done:        make(chan struct{}),
	}
}

// Start starts the watcher. It runs until ctx is cancelled or Stop is called.
func (w *Watcher) Start(ctx context.Context) error {
	w.mu.Lock()
	if w.started {
		w.mu.Unlock()
		return nil
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		w.mu.Unlock()
		return err
	}
	w.watcher = watcher
	w.started = true
	for _, root := range w.roots {
		if err := w.addRootLocked(root); err != nil {
			_ = w.watcher.Close()
			w.watcher = nil
			w.started = false
			w.mu.Unlock()
			return err
		}
	}
	w.mu.Unlock()
	go w.run(ctx)
	return nil
}

func (w *Watcher) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			w.Stop()
			return
		case <-w.done:
			return
		case ev, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			w.handleEvent(ev)
		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			if err != nil {
				// Log in caller; we have no logger here
				_ = err
			}
		}
	}
}

func (w *Watcher) handleEvent(ev fsnotify.Event) {
	path := ev.Name
	if !w.underRoot(path) {
		return
	}
	switch ev.Op {
	case fsnotify.Create, fsnotify.Write:
		if w.matchExtension(path) {
			w.debounceIndex(path)
		}
	case fsnotify.Remove:
		w.cancelDebounce(path)
		if w.matchExtension(path) {
			if w.onRemove != nil {
				w.onRemove(path)
			}
		}
	}
}

func (w *Watcher) underRoot(path string) bool {
	w.mu.Lock()
	roots := append([]string(nil), w.roots...)
	w.mu.Unlock()
	clean := filepath.Clean(path)
	for _, root := range roots {
		rootClean := filepath.Clean(root)
		if rootClean == clean || inDir(rootClean, clean) {
			return true
		}
	}
	return false
}

func inDir(dir, path string) bool {
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func (w *Watcher) matchExtension(path string) bool {
	return matchExtension(path, w.extensions)
}

func matchExtension(path string, extensions []string) bool {
	ext := filepath.Ext(path)
	if len(extensions) == 0 {
		return true
	}
	for _, e := range extensions {
		eNorm := strings.TrimPrefix(strings.ToLower(e), ".")
		extNorm := strings.TrimPrefix(strings.ToLower(ext), ".")
		if eNorm == extNorm {
			return true
		}
	}
	return false
}

func (w *Watcher) debounceIndex(path string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if t, ok := w.debounceMap[path]; ok {
		t.Stop()
	}
	t := time.AfterFunc(w.debounce, func() {
		w.mu.Lock()
		delete(w.debounceMap, path)
		w.mu.Unlock()
		if w.onIndex != nil {
			w.onIndex(path)
		}
	})
	w.debounceMap[path] = t
}

func (w *Watcher) cancelDebounce(path string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if t, ok := w.debounceMap[path]; ok {
		t.Stop()
		delete(w.debounceMap, path)
	}
}

// AddDirectory adds a root directory to watch and optionally syncs existing files.
func (w *Watcher) AddDirectory(root string, syncExisting bool) error {
	abs, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.watcher == nil {
		return nil
	}
	for _, r := range w.roots {
		if filepath.Clean(r) == filepath.Clean(abs) {
			return nil
		}
	}
	if err := w.addRootLocked(abs); err != nil {
		return err
	}
	w.roots = append(w.roots, abs)
	if syncExisting && w.onIndex != nil {
		go w.syncDirectory(abs)
	}
	return nil
}

func (w *Watcher) addRootLocked(root string) error {
	root = filepath.Clean(root)
	if _, err := os.Stat(root); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(root, 0755); err != nil {
				return err
			}
		} else {
			return err
		}
	}
	var paths []string
	add := func(path string, d fs.DirEntry) error {
		if !d.IsDir() {
			return nil
		}
		if err := w.watcher.Add(path); err != nil {
			return err
		}
		paths = append(paths, path)
		return nil
	}
	if w.recursive {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			return add(path, d)
		})
		if err != nil {
			return err
		}
	} else {
		if err := w.watcher.Add(root); err != nil {
			return err
		}
		paths = append(paths, root)
	}
	w.rootPaths[root] = paths
	return nil
}

func (w *Watcher) syncDirectory(root string) {
	w.mu.Lock()
	exts := append([]string(nil), w.extensions...)
	onIndex := w.onIndex
	w.mu.Unlock()
	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if matchExtension(path, exts) {
			if onIndex != nil {
				onIndex(path)
			}
		}
		return nil
	})
}

// RemoveDirectory stops watching the given root. It does not remove indexed documents.
func (w *Watcher) RemoveDirectory(root string) error {
	abs, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	abs = filepath.Clean(abs)
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.watcher == nil {
		return nil
	}
	idx := -1
	for i, r := range w.roots {
		if filepath.Clean(r) == abs {
			idx = i
			break
		}
	}
	if idx < 0 {
		return nil
	}
	paths := w.rootPaths[abs]
	for _, p := range paths {
		_ = w.watcher.Remove(p)
	}
	delete(w.rootPaths, abs)
	w.roots = append(w.roots[:idx], w.roots[idx+1:]...)
	return nil
}

// Directories returns a copy of the current watched root directories.
func (w *Watcher) Directories() []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]string(nil), w.roots...)
}

// SyncExistingFiles indexes all existing files in each watched root that match the configured extensions.
// Call this after Start() to index files that were already present when the watcher started.
func (w *Watcher) SyncExistingFiles() {
	w.mu.Lock()
	roots := append([]string(nil), w.roots...)
	w.mu.Unlock()
	for _, root := range roots {
		w.syncDirectory(root)
	}
}

// Stop stops the watcher and releases resources.
func (w *Watcher) Stop() {
	w.mu.Lock()
	if !w.started || w.watcher == nil {
		w.mu.Unlock()
		return
	}
	for path, t := range w.debounceMap {
		t.Stop()
		delete(w.debounceMap, path)
	}
	_ = w.watcher.Close()
	w.watcher = nil
	w.started = false
	w.mu.Unlock()
	w.stopOnce.Do(func() { close(w.done) })
}
