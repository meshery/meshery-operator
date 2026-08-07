// Package hack holds tests for the release-time scripts in this directory.
// They are exercised through `go test ./...` (and therefore `make test` and CI)
// rather than a separate shell harness, so the release plumbing is covered by
// the same gate as the operator's Go code.
package hack

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// fixtureVersion is the operator release the testdata chart tree currently
// advertises. The tree is a verbatim copy of meshery/meshery master's
// install/kubernetes/helm charts, so the stamp is proven against the real files
// it will meet on the next release rather than a hand-written sketch.
const fixtureVersion = "1.0.5"

// stampVersion is what the tests stamp the fixture to. Deliberately unlike any
// real release so a leftover match cannot be an accident of overlapping digits.
const stampVersion = "9.9.9"

// versionedChartFiles is the complete set of files under the operator chart that
// advertise the operator release, with the line each must carry afterwards.
//
// This map IS the contract the issue reported broken: the stamp used to cover
// only the first three entries, and everything below them drifted - both
// subcharts sat on `stable-latest`, and the parent README advertised 1.0.0 while
// the Chart.yaml beside it read 1.0.5. Deleting an entry here to make a change
// pass is the bug, not the fix.
func versionedChartFiles(version string) map[string][]string {
	return map[string][]string{
		"meshery-operator/Chart.yaml": {
			"version: " + version,
			`appVersion: "` + version + `"`,
		},
		"meshery-operator/values.yaml": {
			`  tag: "` + version + `"`,
		},
		"meshery-operator/README.md": {
			"![Version: " + version + "](https://img.shields.io/badge/Version-" + version + "-",
			"![AppVersion: " + version + "](https://img.shields.io/badge/AppVersion-" + version + "-",
			"| image.tag | string | `\"" + version + "\"`",
		},
		"meshery-operator/charts/meshery-broker/Chart.yaml": {
			`appVersion: "` + version + `"`,
		},
		"meshery-operator/charts/meshery-broker/README.md": {
			"![AppVersion: " + version + "](https://img.shields.io/badge/AppVersion-" + version + "-",
		},
		"meshery-operator/charts/meshery-meshsync/Chart.yaml": {
			`appVersion: "` + version + `"`,
		},
		"meshery-operator/charts/meshery-meshsync/README.md": {
			"![AppVersion: " + version + "](https://img.shields.io/badge/AppVersion-" + version + "-",
		},
	}
}

// helmDir is where the charts live inside a meshery/meshery checkout.
const helmDir = "install/kubernetes/helm"

// newCheckout materialises a throwaway copy of the fixture meshery checkout and
// returns its path.
//
// The parent chart is pre-vendored at the target version so the script's
// `helm dependency update` branch is skipped: that step is unrelated to the
// stamp under test, and depending on it would make these tests need helm on
// PATH and a network. The skip is the script's own documented behaviour for an
// already-vendored version, not a test-only bypass.
func newCheckout(t *testing.T, version string) string {
	t.Helper()

	dir := t.TempDir()
	copyTree(t, filepath.Join("testdata", "meshery"), dir)

	parent := filepath.Join(dir, helmDir, "meshery")
	writeFile(t, filepath.Join(parent, "Chart.lock"),
		"dependencies:\n- name: meshery-operator\n  repository: \"file://../meshery-operator\"\n  version: "+version+"\n")
	writeFile(t, filepath.Join(parent, "charts", "meshery-operator-"+version+".tgz"), "pre-vendored placeholder\n")

	// The script's closing summary asks git whether anything changed. Without a
	// repository that call errors to stderr on every run; with one the test
	// exercises the same reporting branch a release does.
	run(t, dir, "git", "init", "-q")

	return dir
}

// runSync invokes hack/sync-downstream.sh against a checkout, returning its
// combined output and any failure.
func runSync(t *testing.T, checkout, version string) (string, error) {
	t.Helper()

	crds := filepath.Join(t.TempDir(), "crds.yaml")
	writeFile(t, crds, "# rendered CRD bundle placeholder\n")

	script, err := filepath.Abs(filepath.Join("sync-downstream.sh"))
	if err != nil {
		t.Fatalf("resolving script path: %v", err)
	}
	cmd := exec.Command("bash", script, checkout, version)
	cmd.Env = append(os.Environ(), "CRDS_SRC="+crds)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// TestSyncDownstreamStampsEveryVersionedFile is the regression test for
// meshery/meshery-operator#878. It fails against the previous partial-coverage
// stamp, which left both subcharts and every README untouched.
func TestSyncDownstreamStampsEveryVersionedFile(t *testing.T) {
	checkout := newCheckout(t, stampVersion)

	out, err := runSync(t, checkout, stampVersion)
	if err != nil {
		t.Fatalf("sync-downstream.sh failed: %v\n%s", err, out)
	}

	for rel, wantLines := range versionedChartFiles(stampVersion) {
		body := readFile(t, filepath.Join(checkout, helmDir, rel))
		for _, want := range wantLines {
			if !strings.Contains(body, want) {
				t.Errorf("%s: not stamped to %s - missing %q", rel, stampVersion, want)
			}
		}
	}

	// The parent meshery chart's dependency on the operator moves too.
	parent := readFile(t, filepath.Join(checkout, helmDir, "meshery", "Chart.yaml"))
	if !strings.Contains(parent, "- name: meshery-operator\n    version: "+stampVersion) {
		t.Errorf("meshery/Chart.yaml: meshery-operator dependency not bumped to %s", stampVersion)
	}

	// Nothing anywhere in the chart tree may still advertise the old release.
	// A per-file assertion cannot catch a versioned line nobody thought to list;
	// this can, and it is what would have caught the subchart drift years ago.
	for _, rel := range walkFiles(t, filepath.Join(checkout, helmDir)) {
		if strings.HasSuffix(rel, ".tgz") {
			continue // opaque archive; step 3 of the script owns it
		}
		body := readFile(t, filepath.Join(checkout, helmDir, rel))
		for i, line := range strings.Split(body, "\n") {
			if strings.Contains(line, fixtureVersion) {
				t.Errorf("%s:%d still advertises the old release %s: %s",
					rel, i+1, fixtureVersion, strings.TrimSpace(line))
			}
		}
	}
}

// TestSyncDownstreamLeavesSubchartOwnVersionAlone pins the other half of the
// contract: a subchart's `version` tracks the subchart's own lifecycle, not the
// operator release, and the stamp must not touch it. The two live one line apart
// in the same file, so a careless `^version:` rewrite would take both.
func TestSyncDownstreamLeavesSubchartOwnVersionAlone(t *testing.T) {
	checkout := newCheckout(t, stampVersion)

	if out, err := runSync(t, checkout, stampVersion); err != nil {
		t.Fatalf("sync-downstream.sh failed: %v\n%s", err, out)
	}

	for _, subchart := range []string{"meshery-broker", "meshery-meshsync"} {
		base := filepath.Join(checkout, helmDir, "meshery-operator", "charts", subchart)

		if got := readFile(t, filepath.Join(base, "Chart.yaml")); !strings.Contains(got, "version: 0.5.0") {
			t.Errorf("%s/Chart.yaml: subchart's own version was rewritten; it must stay 0.5.0", subchart)
		}
		if got := readFile(t, filepath.Join(base, "README.md")); !strings.Contains(got, "![Version: 0.5.0]") {
			t.Errorf("%s/README.md: subchart's own Version badge was rewritten; it must stay 0.5.0", subchart)
		}
	}
}

// TestSyncDownstreamIsIdempotent pins the property the script's header claims:
// a second run at the same version produces no further changes. The release
// workflow commits whatever the script leaves behind, so a non-idempotent stamp
// would churn meshery/meshery on every re-run.
func TestSyncDownstreamIsIdempotent(t *testing.T) {
	checkout := newCheckout(t, stampVersion)

	if out, err := runSync(t, checkout, stampVersion); err != nil {
		t.Fatalf("first run failed: %v\n%s", err, out)
	}
	first := snapshot(t, filepath.Join(checkout, helmDir))

	if out, err := runSync(t, checkout, stampVersion); err != nil {
		t.Fatalf("second run failed: %v\n%s", err, out)
	}
	second := snapshot(t, filepath.Join(checkout, helmDir))

	for rel, want := range first {
		if got, ok := second[rel]; !ok {
			t.Errorf("%s: removed by the second run", rel)
		} else if got != want {
			t.Errorf("%s: second run changed it; the stamp is not idempotent", rel)
		}
	}
	for rel := range second {
		if _, ok := first[rel]; !ok {
			t.Errorf("%s: created by the second run", rel)
		}
	}
}

// TestSyncDownstreamFailsWhenAStampPatternStopsMatching is the reason the
// assertions exist. A silently-matching-nothing substitution is exactly how the
// README ended up advertising 1.0.0 next to a 1.0.5 Chart.yaml, so a file the
// stamp can no longer reach must fail the release sync rather than pass it.
func TestSyncDownstreamFailsWhenAStampPatternStopsMatching(t *testing.T) {
	for _, tc := range []struct {
		name    string
		rel     string
		old     string
		new     string
		wantMsg string
	}{
		{
			name:    "README image.tag row reformatted",
			rel:     "meshery-operator/README.md",
			old:     "| image.tag | string | `\"" + fixtureVersion + "\"`",
			new:     "| image.tag | str | `\"" + fixtureVersion + "\"`",
			wantMsg: "image.tag row was not stamped",
		},
		{
			name:    "README AppVersion badge reformatted",
			rel:     "meshery-operator/README.md",
			old:     "![AppVersion: " + fixtureVersion + "](https://img.shields.io/badge/AppVersion-",
			new:     "![App Version: " + fixtureVersion + "](https://img.shields.io/badge/AppVersion-",
			wantMsg: "AppVersion badge was not stamped",
		},
		{
			name:    "values.yaml image tag re-indented",
			rel:     "meshery-operator/values.yaml",
			old:     "  tag: \"" + fixtureVersion + "\"",
			new:     "    tag: \"" + fixtureVersion + "\"",
			wantMsg: "image.tag was not stamped",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			checkout := newCheckout(t, stampVersion)
			path := filepath.Join(checkout, helmDir, tc.rel)

			body := readFile(t, path)
			if !strings.Contains(body, tc.old) {
				t.Fatalf("%s: fixture no longer contains %q; refresh testdata from meshery master", tc.rel, tc.old)
			}
			writeFile(t, path, strings.Replace(body, tc.old, tc.new, 1))

			out, err := runSync(t, checkout, stampVersion)
			if err == nil {
				t.Fatalf("expected a failure when %s can no longer be stamped, got success:\n%s", tc.rel, out)
			}
			if !strings.Contains(out, tc.wantMsg) {
				t.Errorf("failure did not name the unstamped value; want %q in:\n%s", tc.wantMsg, out)
			}
		})
	}
}

// TestSyncDownstreamFailsOnUnreachableSubchart covers the discovery loop's blind
// spot: subcharts are found by walking charts/*/, so a declared dependency that
// is not unpacked as a directory is never reached. Passing quietly there would
// reintroduce the drift in a new shape.
func TestSyncDownstreamFailsOnUnreachableSubchart(t *testing.T) {
	checkout := newCheckout(t, stampVersion)
	path := filepath.Join(checkout, helmDir, "meshery-operator", "Chart.yaml")

	body := readFile(t, path)
	writeFile(t, path, strings.Replace(body,
		"dependencies:\n",
		"dependencies:\n  - name: meshery-newthing\n    version: 0.1.0\n", 1))

	out, err := runSync(t, checkout, stampVersion)
	if err == nil {
		t.Fatalf("expected a failure for an undiscoverable subchart, got success:\n%s", out)
	}
	if !strings.Contains(out, "meshery-newthing") {
		t.Errorf("failure did not name the unstamped dependency:\n%s", out)
	}
}

// TestSyncDownstreamRejectsMovingTags guards the repo-wide "no moving image
// tags" rule at the chart boundary. A published Helm archive is immutable, so an
// appVersion of `stable-latest` advertises a different application from one
// publish to the next - which is the state both subcharts were actually found in.
func TestSyncDownstreamRejectsMovingTags(t *testing.T) {
	for _, version := range []string{"stable-latest", "edge-latest", "latest", "v1.0.5"} {
		t.Run(version, func(t *testing.T) {
			checkout := newCheckout(t, stampVersion)
			out, err := runSync(t, checkout, version)
			if err == nil {
				t.Fatalf("expected %q to be rejected, got success:\n%s", version, out)
			}
			if strings.Contains(readFile(t, filepath.Join(checkout, helmDir, "meshery-operator", "Chart.yaml")), version) {
				t.Errorf("%q reached the chart despite the rejection", version)
			}
		})
	}
}

// --- helpers ---

func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	for _, rel := range walkFiles(t, src) {
		target := filepath.Join(dst, rel)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(target), err)
		}
		body, err := os.ReadFile(filepath.Join(src, rel))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		if err := os.WriteFile(target, body, 0o644); err != nil {
			t.Fatalf("write %s: %v", target, err)
		}
	}
}

// walkFiles lists every regular file under root, as slash-separated paths
// relative to it. .git is skipped so the throwaway repository created by
// newCheckout does not leak into the assertions.
func walkFiles(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		out = append(out, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		t.Fatalf("walking %s: %v", root, err)
	}
	return out
}

// snapshot maps every file under root to its contents, for change detection.
func snapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	out := make(map[string]string)
	for _, rel := range walkFiles(t, root) {
		out[rel] = readFile(t, filepath.Join(root, rel))
	}
	return out
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(body)
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func run(t *testing.T, dir, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s %s: %v\n%s", name, strings.Join(args, " "), err, out)
	}
}
