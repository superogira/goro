// Command mappack builds per-map GRF packs for the web client.
//
// For every map (data/<name>.gat) it resolves the map's full dependency
// closure — the .gat/.gnd/.rsw triple, every model referenced by the .rsw
// (.rsm/.rsm2) with their textures, the .gnd ground textures, and the
// 32-frame water animation — stages those files under their data/ paths,
// and packs them with res.PackGRF into <out>/<name>.grf.
//
// At runtime the client downloads map_pack/<name>.grf before reading
// the map (res.Manager.EnsureWebMapPack); anything the pack misses falls
// back to loose-file streaming, so partial or absent packs are harmless.
//
//	usage: mappack [-data DIR] [-out DIR] [-maps a,b,c] [-workers N] [-force] [-v]
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"unicode/utf8"

	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/transform"

	"github.com/kivutar/goro/res"
)

type mapPackResult struct {
	base    string
	files   int
	bytes   int64
	err     error
	skipped bool
}

func main() {
	dataDir := flag.String("data", "data", "source data directory holding the loose files")
	outDir := flag.String("out", "", "output directory (default map_pack/ beside -data)")
	mapsFlag := flag.String("maps", "", "comma-separated map names (default: every .gat in -data)")
	workers := flag.Int("workers", runtime.NumCPU(), "parallel pack builders")
	force := flag.Bool("force", false, "rebuild packs that already exist")
	verbose := flag.Bool("v", false, "print every resolved dependency")
	flag.Parse()

	out := *outDir
	if out == "" {
		out = filepath.Join(filepath.Dir(filepath.Clean(*dataDir)), "map_pack")
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		fatal(err)
	}

	names, err := mapNames(*dataDir, *mapsFlag)
	if err != nil {
		fatal(err)
	}

	jobs := make(chan int)
	results := make([]mapPackResult, len(names))
	varwg := &sync.WaitGroup{}
	var done atomic.Int64
	for i := 0; i < clampInt(*workers, 1, 16); i++ {
		varwg.Add(1)
		go func() {
			defer varwg.Done()
			for idx := range jobs {
				r := packMap(*dataDir, out, names[idx], *force, *verbose)
				results[idx] = r // per-index writes: no two workers share an index
				n := done.Add(1)
				fmt.Printf("[%d/%d] %s: %s\n", n, len(names), r.base, describe(r))
			}
		}()
	}
	for idx := range names {
		jobs <- idx
	}
	close(jobs)
	varwg.Wait()

	var totalFiles, totalBytes int64
	var failed, skipped, packed int
	for _, r := range results {
		if r.err != nil {
			failed++
			fmt.Fprintf(os.Stderr, "mappack: %s: %v\n", r.base, r.err)
		} else if r.skipped {
			skipped++
		} else {
			packed++
		}
		totalFiles += int64(r.files)
		totalBytes += int64(r.bytes)
	}
	fmt.Printf("packed %d maps (%d files, %s), %d skipped, %d failed -> %s\n",
		packed, totalFiles, humanBytes(totalBytes), skipped, failed, out)
	if failed > 0 {
		os.Exit(1)
	}
}

func describe(r mapPackResult) string {
	if r.skipped {
		return "exists (use -force to rebuild)"
	}
	if r.err != nil {
		return r.err.Error()
	}
	return fmt.Sprintf("%d files, %s", r.files, humanBytes(r.bytes))
}

func humanBytes(n int64) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(n)/(1<<10))
	default:
		return fmt.Sprintf("%d B", n)
	}
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "mappack: %v\n", err)
	os.Exit(1)
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func mapNames(dataDir, mapsFlag string) ([]string, error) {
	if mapsFlag != "" {
		names := strings.Split(mapsFlag, ",")
		for i := range names {
			names[i] = strings.TrimSuffix(strings.TrimSpace(names[i]), ".gat")
		}
		return names, nil
	}
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".gat") {
			continue
		}
		names = append(names, strings.TrimSuffix(entry.Name(), ".gat"))
	}
	sort.Strings(names)
	return names, nil
}

// packMap resolves one map's dependencies and writes its GRF.
func packMap(dataDir, outDir, base string, force, verbose bool) mapPackResult {
	target := filepath.Join(outDir, base+".grf")
	if !force {
		if _, err := os.Stat(target); err == nil {
			return mapPackResult{base: base, skipped: true}
		}
	}

	files, err := mapDependencies(dataDir, base, verbose)
	if err != nil {
		return mapPackResult{base: base, err: err}
	}
	if len(files) == 0 {
		return mapPackResult{base: base, err: fmt.Errorf("no dependencies resolved")}
	}

	staging, err := os.MkdirTemp("", "mappack-")
	if err != nil {
		return mapPackResult{base: base, err: err}
	}
	defer os.RemoveAll(staging)
	for rel, source := range files {
		dst := filepath.Join(staging, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return mapPackResult{base: base, err: err}
		}
		if err := copyFile(dst, source); err != nil {
			return mapPackResult{base: base, err: err}
		}
	}
	stats, err := res.PackGRF(target, staging)
	if err != nil {
		return mapPackResult{base: base, err: err}
	}
	return mapPackResult{base: base, files: stats.Files, bytes: stats.Bytes}
}

// mapDependencies walks the map's reference graph and returns every file
// that exists, keyed by the data/-relative path the client queries.
func mapDependencies(dataDir, base string, verbose bool) (map[string]string, error) {
	files := make(map[string]string)
	add := func(rel string) bool {
		source := filepath.Join(dataDir, filepath.FromSlash(rel))
		if _, err := os.Stat(source); err != nil {
			return false
		}
		files["data/"+rel] = source
		if verbose {
			fmt.Printf("  %s: %s\n", base, rel)
		}
		return true
	}
	resolve := func(candidates []string) bool {
		for _, candidate := range candidates {
			if add(relFromCandidate(candidate)) {
				return true
			}
		}
		return false
	}

	add(base + ".gat")
	if !resolve([]string{"data\\" + base + ".gnd", "data/" + base + ".gnd"}) {
		return files, fmt.Errorf("gnd missing for %s", base)
	}
	resolve([]string{"data\\" + base + ".rsw", "data/" + base + ".rsw"})

	gndData, err := os.ReadFile(filepath.Join(dataDir, base+".gnd"))
	if err != nil {
		return files, err
	}
	gnd, err := res.ParseGND(gndData)
	if err != nil {
		return files, fmt.Errorf("parse gnd: %w", err)
	}
	for _, name := range gnd.Textures {
		resolve(res.GroundTextureCandidates(decodeName(name)))
	}

	rswData, err := os.ReadFile(filepath.Join(dataDir, base+".rsw"))
	if err != nil {
		// No RSW: ground textures above still apply.
		return files, nil
	}
	rsw, err := res.ParseRSW(rswData)
	if err != nil {
		return files, fmt.Errorf("parse rsw: %w", err)
	}
	for _, placement := range rsw.Models {
		if placement.Filename == "" {
			continue
		}
		rsmPath := ""
		for _, candidate := range res.RSMModelCandidates(decodeName(placement.Filename)) {
			rel := relFromCandidate(candidate)
			source := filepath.Join(dataDir, filepath.FromSlash(rel))
			if _, err := os.Stat(source); err == nil {
				rsmPath = source
				add(rel)
				break
			}
		}
		if rsmPath == "" {
			continue
		}
		rsmData, err := os.ReadFile(rsmPath)
		if err != nil {
			continue
		}
		rsm, err := res.ParseRSM(rsmData)
		if err != nil {
			continue // model still ships; its textures fall back to streaming
		}
		for _, name := range rsm.Textures {
			resolve(res.GroundTextureCandidates(decodeName(name)))
		}
	}

	waterType := int(rsw.Water.Type)
	if gnd.Water.Present {
		waterType = int(gnd.Water.Type)
	}
	for frame := 0; frame < 32; frame++ {
		resolve(res.WaterTextureCandidates(waterType, frame))
	}
	return files, nil
}

// relFromCandidate turns a client resource candidate (data\texture\x,
// texture/x, ...) into the data/-relative path used on disk and in packs.
func relFromCandidate(candidate string) string {
	candidate = strings.ReplaceAll(candidate, "\\", "/")
	for strings.HasPrefix(candidate, "./") {
		candidate = candidate[2:]
	}
	candidate = strings.TrimPrefix(candidate, "data/")
	return candidate
}

// decodeName converts the CP949 byte strings carried by GRF-era map files
// into the UTF-8 spelling the client fetches (fs_web's euckrVariant) and
// the data/ tree uses on disk. Already-valid UTF-8 names pass through.
func decodeName(name string) string {
	if name == "" || utf8.ValidString(name) {
		return name
	}
	decoded, _, err := transform.Bytes(korean.EUCKR.NewDecoder(), []byte(name))
	if err != nil {
		return name
	}
	return string(decoded)
}

func copyFile(dst, src string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		return err
	}
	return os.Chtimes(dst, info.ModTime(), info.ModTime())
}
