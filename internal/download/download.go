// Package download fetches files concurrently with checksum verification.
package download

import (
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// UserAgent identifies the launcher to web APIs (Modrinth requires one).
const UserAgent = "Rabbit-Launcher (+https://github.com/vellnico/Rabbit)"

var client = &http.Client{
	Timeout: 5 * time.Minute,
	Transport: &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		MaxIdleConnsPerHost: 32,
		IdleConnTimeout:     60 * time.Second,
	},
}

// File is a single file to fetch. Hash is a sha1, sha256 or sha512 hex
// checksum (told apart by length); empty means "trust whatever the server sends".
type File struct {
	URL  string
	Path string
	Hash string
	Size int64
}

// Progress is reported while a batch is downloading.
type Progress func(done, total int)

// Do sends a request with the launcher's user agent and the shared client,
// whose timeout keeps a stalled server from hanging the launcher.
func Do(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", UserAgent)
	return client.Do(req)
}

// Get performs a GET request with the launcher's user agent.
func Get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	return resp, nil
}

// JSON fetches url and decodes it into v.
func JSON(ctx context.Context, url string, v any) error {
	resp, err := Get(ctx, url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(v)
}

// Valid reports whether the file on disk matches the expected size and hash.
func Valid(f File) bool {
	st, err := os.Stat(f.Path)
	if err != nil || st.IsDir() {
		return false
	}
	if f.Size > 0 && st.Size() != f.Size {
		return false
	}
	if f.Hash == "" {
		return true
	}
	sum, err := hashFile(f.Path, f.Hash)
	return err == nil && strings.EqualFold(sum, f.Hash)
}

func newHash(expected string) hash.Hash {
	switch len(expected) {
	case 64:
		return sha256.New()
	case 128:
		return sha512.New()
	}
	return sha1.New()
}

func hashFile(path, expected string) (string, error) {
	fh, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer fh.Close()
	h := newHash(expected)
	if _, err := io.Copy(h, fh); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// One downloads a single file, skipping it if it is already valid.
func One(ctx context.Context, f File) error {
	if Valid(f) {
		return nil
	}
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		if err = fetch(ctx, f); err == nil || ctx.Err() != nil {
			return err
		}
		select { // a cancelled launch must not wait out the retries
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(attempt+1) * 500 * time.Millisecond):
		}
	}
	return err
}

func fetch(ctx context.Context, f File) error {
	resp, err := Get(ctx, f.URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if err := os.MkdirAll(filepath.Dir(f.Path), 0o755); err != nil {
		return err
	}
	tmp := f.Path + ".part"
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	h := newHash(f.Hash)
	_, err = io.Copy(io.MultiWriter(out, h), resp.Body)
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(tmp)
		return err
	}
	if f.Hash != "" {
		if sum := hex.EncodeToString(h.Sum(nil)); !strings.EqualFold(sum, f.Hash) {
			os.Remove(tmp)
			return fmt.Errorf("%s: checksum mismatch", filepath.Base(f.Path))
		}
	}
	return os.Rename(tmp, f.Path)
}

// All downloads files using a pool of workers and reports progress.
func All(parent context.Context, files []File, workers int, progress Progress) error {
	if len(files) == 0 {
		return nil
	}
	ctx, cancel := context.WithCancel(parent)
	defer cancel()

	jobs := make(chan File)
	var (
		wg       sync.WaitGroup
		done     atomic.Int32
		errOnce  sync.Once
		firstErr error
	)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for f := range jobs {
				if err := One(ctx, f); err != nil {
					errOnce.Do(func() { firstErr = err; cancel() })
					continue
				}
				n := done.Add(1)
				if progress != nil {
					progress(int(n), len(files))
				}
			}
		}()
	}
loop:
	for _, f := range files {
		select {
		case jobs <- f:
		case <-ctx.Done():
			break loop
		}
	}
	close(jobs)
	wg.Wait()
	if firstErr != nil {
		return firstErr
	}
	return parent.Err()
}
