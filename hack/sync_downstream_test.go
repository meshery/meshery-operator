// Package hack holds tests for the release-time scripts in this directory.
// They are exercised through `go test ./...` (and therefore `make test` and CI)
// rather than a separate shell harness, so the release plumbing is covered by
// the same gate as the operator's Go code.
package hack

import (
	"maps"
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

// prereleaseVersion is stampVersion's release candidate. The semver guard admits
// prereleases, so every stamped value has to round-trip a version containing a
// '-' - which is also the character shields.io reads as a badge field separator.
const prereleaseVersion = "9.9.9-rc.1"

// badgeVersion is the version as it appears inside a shields.io badge path,
// where a '-' belonging to the value must be doubled to survive the
// label-message-colour split. Mirrors what helm-docs emits.
func badgeVersion(version string) string {
	return strings.ReplaceAll(version, "-", "--")
}

// parentChartReadme is the operator chart's own README, relative to the helm
// directory. It is the file the issue caught advertising 1.0.0 beside a 1.0.5
// Chart.yaml, so it turns up both in the contract map and among the cases that
// pin what happens when a stamp pattern stops matching.
const parentChartReadme = "meshery-operator/README.md"

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
		parentChartReadme: {
			"![Version: " + version + "](https://img.shields.io/badge/Version-" + badgeVersion(version) + "-informational",
			"![AppVersion: " + version + "](https://img.shields.io/badge/AppVersion-" + badgeVersion(version) + "-informational",
			"| image.tag | string | `\"" + version + "\"`",
		},
		"meshery-operator/charts/meshery-broker/Chart.yaml": {
			`appVersion: "` + version + `"`,
		},
		"meshery-operator/charts/meshery-broker/README.md": {
			"![AppVersion: " + version + "](https://img.shields.io/badge/AppVersion-" + badgeVersion(version) + "-informational",
		},
		"meshery-operator/charts/meshery-meshsync/Chart.yaml": {
			`appVersion: "` + version + `"`,
		},
		"meshery-operator/charts/meshery-meshsync/README.md": {
			"![AppVersion: " + version + "](https://img.shields.io/badge/AppVersion-" + badgeVersion(version) + "-informational",
		},
	}
}

// helmDir is where the charts live inside a meshery/meshery checkout.
const helmDir = "install/kubernetes/helm"

// newCheckout materialises a throwaway copy of the fixture meshery checkout and
// returns its path.
//
// The parent chart starts out pre-vendored at the target version. That is the
// precondition the script's step-3 skip used to key on, and it is what makes
// TestSyncDownstreamReVendorsOnContentChange meaningful: the re-vendor there can
// only have been driven by the chart's content changing, never by the version.
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

// helmCallLog is where the stub records its invocations, at the checkout root so
// it stays outside the chart tree the assertions walk.
const helmCallLog = ".helm-calls"

// writeHelmStub installs a stand-in for `helm dependency update` inside the
// checkout and returns its path.
//
// The tests stay hermetic - no helm on PATH, no network - but step 3's decision
// to re-vendor is behaviour worth asserting rather than avoiding, so the stub
// records every invocation and reproduces the two side effects the script's own
// skip condition reads back: the vendored archive and the Chart.lock entry.
func writeHelmStub(t *testing.T, checkout string) string {
	t.Helper()

	path := filepath.Join(checkout, ".helm-stub")
	writeFile(t, path, `#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >> "$HELM_STUB_LOG"
chart="$3"
mkdir -p "$chart/charts"
printf 'packaged meshery-operator %s\n' "$HELM_STUB_VERSION" \
  > "$chart/charts/meshery-operator-$HELM_STUB_VERSION.tgz"
printf 'dependencies:\n- name: meshery-operator\n  repository: "file://../meshery-operator"\n  version: %s\n' \
  "$HELM_STUB_VERSION" > "$chart/Chart.lock"
`)
	if err := os.Chmod(path, 0o755); err != nil {
		t.Fatalf("chmod %s: %v", path, err)
	}
	return path
}

// helmCalls returns the arguments of every re-vendor the script has performed
// against this checkout, across all runs so far.
func helmCalls(t *testing.T, checkout string) []string {
	t.Helper()

	body, err := os.ReadFile(filepath.Join(checkout, helmCallLog))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatalf("reading %s: %v", helmCallLog, err)
	}
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "\n")
}

// runSync invokes hack/sync-downstream.sh against a checkout, returning its
// combined output and any failure.
func runSync(t *testing.T, checkout, version string) (string, error) {
	t.Helper()

	crds := filepath.Join(t.TempDir(), "crds.yaml")
	writeFile(t, crds, "# rendered CRD bundle placeholder\n")

	script, err := filepath.Abs("sync-downstream.sh")
	if err != nil {
		t.Fatalf("resolving script path: %v", err)
	}
	cmd := exec.Command("bash", script, checkout, version)
	cmd.Env = append(os.Environ(),
		"CRDS_SRC="+crds,
		"HELM_BIN="+writeHelmStub(t, checkout),
		"HELM_STUB_LOG="+filepath.Join(checkout, helmCallLog),
		"HELM_STUB_VERSION="+version,
	)
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

	assertStampedTo(t, checkout, stampVersion)
	assertNoFileAdvertises(t, checkout, fixtureVersion)
}

// assertStampedTo checks every value in the contract map, plus the parent
// meshery chart's dependency on the operator.
func assertStampedTo(t *testing.T, checkout, version string) {
	t.Helper()

	for rel, wantLines := range versionedChartFiles(version) {
		body := readFile(t, filepath.Join(checkout, helmDir, rel))
		for _, want := range wantLines {
			if !strings.Contains(body, want) {
				t.Errorf("%s: not stamped to %s - missing %q", rel, version, want)
			}
		}
	}

	parent := readFile(t, filepath.Join(checkout, helmDir, "meshery", "Chart.yaml"))
	if !strings.Contains(parent, "- name: meshery-operator\n    version: "+version) {
		t.Errorf("meshery/Chart.yaml: meshery-operator dependency not bumped to %s", version)
	}
}

// assertNoFileAdvertises sweeps the whole chart tree for a version that should
// no longer appear anywhere in it. A per-file assertion cannot catch a versioned
// line nobody thought to list; this can, and it is what would have caught the
// subchart drift years ago.
//
// The fixture is trimmed by a rule independent of what the stamp touches (every
// non-template file under the operator chart), so this sweep can surface a drift
// site nobody has thought of yet rather than only re-confirm the known ones.
func assertNoFileAdvertises(t *testing.T, checkout, version string) {
	t.Helper()

	for _, rel := range walkFiles(t, filepath.Join(checkout, helmDir)) {
		if strings.HasSuffix(rel, ".tgz") {
			continue // opaque archive; step 3 of the script owns it
		}
		body := readFile(t, filepath.Join(checkout, helmDir, rel))
		for i, line := range strings.Split(body, "\n") {
			if strings.Contains(line, version) {
				t.Errorf("%s:%d still advertises %s: %s",
					rel, i+1, version, strings.TrimSpace(line))
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
//
// Run over a prerelease as well as a plain release: '-' is the shields.io badge
// field separator, so a badge rewrite that stops at the first one appended the
// prerelease suffix again on every pass while every plain-version case stayed
// clean.
func TestSyncDownstreamIsIdempotent(t *testing.T) {
	for _, version := range []string{stampVersion, prereleaseVersion} {
		t.Run(version, func(t *testing.T) {
			checkout := newCheckout(t, version)

			if out, err := runSync(t, checkout, version); err != nil {
				t.Fatalf("first run failed: %v\n%s", err, out)
			}
			first := snapshot(t, filepath.Join(checkout, helmDir))

			if out, err := runSync(t, checkout, version); err != nil {
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
		})
	}
}

// TestSyncDownstreamStampsAPrereleaseThenTheFinalRelease walks the sequence a
// release candidate actually takes: stamp the prerelease, then stamp the final
// release over the same tree. Both halves have to be exact - a badge rewrite
// that consumes only up to the first '-' leaves `Version-9.9.9-rc.1-rc.1-` after
// the first step and a stale `-rc.1` after the second, while the old
// prefix-only assertion accepted each of them.
func TestSyncDownstreamStampsAPrereleaseThenTheFinalRelease(t *testing.T) {
	checkout := newCheckout(t, prereleaseVersion)

	if out, err := runSync(t, checkout, prereleaseVersion); err != nil {
		t.Fatalf("prerelease stamp failed: %v\n%s", err, out)
	}
	assertStampedTo(t, checkout, prereleaseVersion)
	assertNoFileAdvertises(t, checkout, fixtureVersion)

	// The final release goes over the top of the candidate. Nothing may keep the
	// candidate's suffix, in a badge or anywhere else.
	if out, err := runSync(t, checkout, stampVersion); err != nil {
		t.Fatalf("final release stamp failed: %v\n%s", err, out)
	}
	assertStampedTo(t, checkout, stampVersion)
	assertNoFileAdvertises(t, checkout, "rc.1")
}

// TestSyncDownstreamReVendorsOnContentChange pins step 3's trigger: the vendored
// archive folds in every file the stamp rewrites, so "already vendored at this
// version" is not enough to skip repackaging it. A re-sync of an existing tag
// (the workflow's supported workflow_dispatch path) would otherwise commit
// freshly stamped sources beside an archive still carrying the old ones.
func TestSyncDownstreamReVendorsOnContentChange(t *testing.T) {
	// Vendored at the target version already, but its subcharts and READMEs still
	// advertise the previous release - so only a content signal can trigger this.
	checkout := newCheckout(t, stampVersion)

	if out, err := runSync(t, checkout, stampVersion); err != nil {
		t.Fatalf("first run failed: %v\n%s", err, out)
	}
	// Assert the invocation, not just that one happened: the count alone would
	// stay green if step 3 regressed to a wrong subcommand or re-vendored the
	// wrong chart, which is exactly the contract worth pinning here.
	wantCall := "dependency update " + filepath.Join(checkout, helmDir, "meshery")
	calls := helmCalls(t, checkout)
	if len(calls) != 1 {
		t.Fatalf("a stamp that rewrote the chart must re-vendor it exactly once; helm calls: %v", calls)
	}
	if calls[0] != wantCall {
		t.Errorf("re-vendored with %q, want %q", calls[0], wantCall)
	}

	// A genuine no-op re-run must still skip, or every re-sync churns the
	// Chart.lock timestamp and the archive bytes for nothing.
	if out, err := runSync(t, checkout, stampVersion); err != nil {
		t.Fatalf("second run failed: %v\n%s", err, out)
	}
	if calls := helmCalls(t, checkout); len(calls) != 1 {
		t.Errorf("a no-op re-run re-vendored; step 3 must skip an unchanged chart; helm calls: %v", calls)
	}
}

// TestSyncDownstreamReVendorsOnVersionChange walks the release path a version
// bump takes. It does NOT isolate step 3's version clause: a bump rewrites every
// file in the contract map, so the content clause short-circuits the decision
// first. What it pins is the end-to-end outcome around the repackage - the
// archive vendored under the old name is gone and the new name is in place, so a
// bump cannot leave two archives behind for helm to choose between. The clauses
// the content signal hides are isolated in
// TestSyncDownstreamReVendorsOnMissingOrStaleArtifact.
func TestSyncDownstreamReVendorsOnVersionChange(t *testing.T) {
	checkout := newCheckout(t, stampVersion)

	if out, err := runSync(t, checkout, stampVersion); err != nil {
		t.Fatalf("first run failed: %v\n%s", err, out)
	}
	before := len(helmCalls(t, checkout))

	if out, err := runSync(t, checkout, prereleaseVersion); err != nil {
		t.Fatalf("version bump failed: %v\n%s", err, out)
	}
	if got := len(helmCalls(t, checkout)) - before; got != 1 {
		t.Errorf("a version bump must re-vendor exactly once, got %d further helm calls", got)
	}

	vendored := filepath.Join(checkout, helmDir, "meshery", "charts")
	if _, err := os.Stat(filepath.Join(vendored, "meshery-operator-"+prereleaseVersion+".tgz")); err != nil {
		t.Errorf("the bumped version was not vendored: %v", err)
	}
	if _, err := os.Stat(filepath.Join(vendored, "meshery-operator-"+stampVersion+".tgz")); !os.IsNotExist(err) {
		t.Errorf("the archive from the previous version survived the bump; two archives leave helm a choice")
	}
}

// TestSyncDownstreamReVendorsOnMissingOrStaleArtifact isolates the two clauses
// of step 3's trigger that a content change hides. The chart is stamped once so
// its content settles; every run after that rewrites the same bytes, so the
// content clause cannot fire and only the artifact state can decide. That is the
// repair path for a tree whose sources are stamped at V but whose vendored
// artifact never landed - a half-finished sync, or a hand-reverted archive - and
// without it the next sync would see settled content and skip forever.
//
// Deleting either clause from the script must fail this test; that was checked
// by deleting them.
func TestSyncDownstreamReVendorsOnMissingOrStaleArtifact(t *testing.T) {
	// disturb precedes name so the struct packs its two pointers adjacently
	// (govet fieldalignment); the reader-friendly order is the other way round.
	for _, tc := range []struct {
		disturb func(t *testing.T, parent string)
		name    string
	}{
		{
			disturb: func(t *testing.T, parent string) {
				t.Helper()
				archive := filepath.Join(parent, "charts", "meshery-operator-"+stampVersion+".tgz")
				if err := os.Remove(archive); err != nil {
					t.Fatalf("removing the vendored archive: %v", err)
				}
			},
			name: "vendored archive missing",
		},
		{
			disturb: func(t *testing.T, parent string) {
				t.Helper()
				writeFile(t, filepath.Join(parent, "Chart.lock"),
					"dependencies:\n- name: meshery-operator\n  repository: \"file://../meshery-operator\"\n  version: 0.0.1\n")
			},
			name: "Chart.lock names another version",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			checkout := newCheckout(t, stampVersion)
			operatorChart := filepath.Join(checkout, helmDir, "meshery-operator")

			if out, err := runSync(t, checkout, stampVersion); err != nil {
				t.Fatalf("first run failed: %v\n%s", err, out)
			}
			settled := snapshot(t, operatorChart)
			before := len(helmCalls(t, checkout))

			// Only the parent chart is disturbed. Nothing under the operator chart
			// moves, so the content clause is out of the picture by construction.
			tc.disturb(t, filepath.Join(checkout, helmDir, "meshery"))

			if out, err := runSync(t, checkout, stampVersion); err != nil {
				t.Fatalf("repair run failed: %v\n%s", err, out)
			}
			if got := len(helmCalls(t, checkout)) - before; got != 1 {
				t.Errorf("step 3 must repair this exactly once, got %d further helm calls", got)
			}
			if !maps.Equal(snapshot(t, operatorChart), settled) {
				t.Error("the repair run rewrote the operator chart, so the content clause could have " +
					"decided the re-vendor and this case no longer isolates the artifact state")
			}
		})
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
			rel:     parentChartReadme,
			old:     "| image.tag | string | `\"" + fixtureVersion + "\"`",
			new:     "| image.tag | str | `\"" + fixtureVersion + "\"`",
			wantMsg: "image.tag row was not stamped",
		},
		{
			name:    "README AppVersion badge reformatted",
			rel:     parentChartReadme,
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
		{
			// The one substitution whose silent no-op leaves no trace: with a
			// non-exact constraint helm resolves the stale entry and vendors the
			// PREVIOUS operator archive without complaint, so the sync "succeeds"
			// having shipped the last release.
			name:    "parent dependency entry lost its version key",
			rel:     "meshery/Chart.yaml",
			old:     "  - name: meshery-operator\n    version: " + fixtureVersion + "\n",
			new:     "  - name: meshery-operator\n",
			wantMsg: "meshery-operator dependency version was not stamped",
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

// TestSyncDownstreamFailsOnUndeclaredSubchart covers the other direction of the
// same cross-check. The walk stamps the operator release onto every chart it
// finds under charts/, so a chart that does not track that release - a vendored
// upstream NATS chart, say, mirroring what this repo keeps under
// pkg/broker/chart - would be relabelled to advertise an application it does not
// install. Skipping it quietly would be no better: that silence is how the
// declared-but-unreachable direction rotted for releases on end.
func TestSyncDownstreamFailsOnUndeclaredSubchart(t *testing.T) {
	checkout := newCheckout(t, stampVersion)

	// Sorts after both declared subcharts, so the run reaches it only once the
	// legitimate ones are stamped - the position where a silent pass is likeliest.
	intruder := filepath.Join(checkout, helmDir, "meshery-operator", "charts", "nats")
	const intruderChart = "apiVersion: v2\nname: nats\nversion: 1.3.10\nappVersion: \"2.11.10\"\n"
	writeFile(t, filepath.Join(intruder, "Chart.yaml"), intruderChart)

	out, err := runSync(t, checkout, stampVersion)
	if err == nil {
		t.Fatalf("expected a failure for an undeclared subchart, got success:\n%s", out)
	}
	if !strings.Contains(out, "nats") || !strings.Contains(out, "not a declared dependency") {
		t.Errorf("failure did not name the undeclared subchart and what to do about it:\n%s", out)
	}
	if got := readFile(t, filepath.Join(intruder, "Chart.yaml")); got != intruderChart {
		t.Errorf("the undeclared subchart was rewritten before the sync refused it:\n%s", got)
	}
}

// TestSyncDownstreamRejectsMovingTags guards the repo-wide "no moving image
// tags" rule at the chart boundary. A published Helm archive is immutable, so an
// appVersion of `stable-latest` advertises a different application from one
// publish to the next - which is the state both subcharts were actually found in.
func TestSyncDownstreamRejectsMovingTags(t *testing.T) {
	for _, version := range []string{"stable-latest", "edge-latest", "latest", "v1.0.5"} {
		t.Run(version, func(t *testing.T) {
			assertVersionRejected(t, version)
		})
	}
}

// TestSyncDownstreamRejectsMalformedVersions covers the other half of the guard.
// Moving tags are the failure it was written for, but a value can be neither a
// moving tag nor a version - and a loose "digits, dots and a suffix" pattern
// waves those through. They are worse than a moving tag in one respect: the
// stamp succeeds and the chart is written, so the first sign of trouble is helm
// or an image pull failing opaquely on a version that never existed.
func TestSyncDownstreamRejectsMalformedVersions(t *testing.T) {
	for _, version := range []string{
		"1.0",         // too few components
		"1.0.5.1",     // too many
		"1.0.5-",      // prerelease marker with no identifier
		"1.0.5-.",     // empty prerelease identifiers
		"1.0.5-rc..1", // empty identifier between dots
		"01.2.3",      // leading zero in a numeric identifier
		"1.02.3",      //
		"1.2.03",      //
		"1.2.3-01",    // leading zero in a numeric prerelease identifier
		"1.2.3+build", // build metadata is not legal in a container image tag
		"",            // empty
		"1.2.x",       // non-numeric component
	} {
		t.Run(version, func(t *testing.T) {
			assertVersionRejected(t, version)
		})
	}
}

// TestSyncDownstreamAcceptsValidSemver is the guard's other side: tightening a
// pattern is only safe if it still admits every version a release can legitimately
// carry, so the prerelease forms the release process can produce are pinned too.
func TestSyncDownstreamAcceptsValidSemver(t *testing.T) {
	for _, version := range []string{"1.0.5", "0.0.0", "10.20.30", "1.2.3-rc.1", "1.2.3-0.3.7", "1.2.3-alpha-1"} {
		t.Run(version, func(t *testing.T) {
			checkout := newCheckout(t, version)
			if out, err := runSync(t, checkout, version); err != nil {
				t.Fatalf("valid version %q was rejected: %v\n%s", version, err, out)
			}
			assertStampedTo(t, checkout, version)
		})
	}
}

// assertVersionRejected runs a stamp that must fail, and checks that the whole
// chart tree was left alone. Refusing is only half of it: a guard that rejected
// the version after writing some of the files would leave exactly the
// half-stamped state this script exists to prevent.
//
// The comparison covers every file, not just the parent Chart.yaml: the stamp
// rewrites values.yaml, three READMEs, both subcharts and the vendored
// dependency chart, so a partial write anywhere else would pass a single-file
// check while leaving the tree inconsistent.
func assertVersionRejected(t *testing.T, version string) {
	t.Helper()

	checkout := newCheckout(t, stampVersion)
	tree := filepath.Join(checkout, helmDir)
	before := snapshot(t, tree)

	out, err := runSync(t, checkout, version)
	if err == nil {
		t.Fatalf("expected %q to be rejected, got success:\n%s", version, out)
	}

	after := snapshot(t, tree)
	for rel, want := range before {
		got, ok := after[rel]
		if !ok {
			t.Errorf("%q was rejected but %s was removed", version, rel)
			continue
		}
		if got != want {
			t.Errorf("%q was rejected but %s was already rewritten:\n%s", version, rel, got)
		}
	}
	for rel := range after {
		if _, ok := before[rel]; !ok {
			t.Errorf("%q was rejected but %s was created", version, rel)
		}
	}
}

// TestReleaseVersionGuardIsShared pins that both release scripts enforce the
// same version set - the whole reason the check lives in hack/lib.
//
// It drives hack/stamp-operator-version.sh directly rather than only
// sync-downstream.sh. Only the rejection path is exercised against the real
// script: the guard runs before that script touches anything, so an invalid
// version is safe, while a valid one would rewrite this repository's own
// Makefile, config/ and bundle/. Acceptance is covered against the sourced
// helper itself, which is the same code both scripts run.
func TestReleaseVersionGuardIsShared(t *testing.T) {
	// argsFor puts the version where each script expects it. sync-downstream
	// takes the checkout path first; the guard runs before that path is read, so
	// a name that does not exist is fine and keeps this test off the filesystem.
	scripts := map[string]func(version string) []string{
		"sync-downstream.sh":        func(v string) []string { return []string{"no-such-checkout", v} },
		"stamp-operator-version.sh": func(v string) []string { return []string{v} },
	}

	// Both scripts must actually source the shared guard. Without this, a future
	// change could reintroduce a local copy and the drift the shared file exists
	// to prevent would be back with every test still green.
	for script := range scripts {
		if body := readFile(t, script); !strings.Contains(body, "lib/release-version.sh") {
			t.Errorf("%s does not source the shared release-version guard", script)
		}
	}

	for _, version := range []string{"stable-latest", "v1.0.5", "01.2.3", "1.2.3-01", "1.0.5-.", "1.0"} {
		t.Run("reject/"+version, func(t *testing.T) {
			for script, argsFor := range scripts {
				out, err := exec.Command("bash", append([]string{script}, argsFor(version)...)...).CombinedOutput() //nolint:gosec // fixed script list
				if err == nil {
					t.Errorf("%s accepted %q:\n%s", script, version, out)
				}
				if !strings.Contains(string(out), version) {
					t.Errorf("%s rejected %q without naming it:\n%s", script, version, out)
				}
			}
		})
	}

	for _, version := range []string{"1.0.5", "0.0.0", "1.2.3-rc.1", "1.2.3-0.3.7"} {
		t.Run("accept/"+version, func(t *testing.T) {
			out, err := exec.Command("bash", "-c",
				`. lib/release-version.sh; require_release_version "$1" "test"`, "_", version).CombinedOutput()
			if err != nil {
				t.Errorf("the shared guard rejected the valid version %q: %v\n%s", version, err, out)
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
