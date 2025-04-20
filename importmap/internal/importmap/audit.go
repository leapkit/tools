package importmap

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

func (m *manager) Audit(ctx context.Context) error {
	pkgReg := regexp.MustCompile(`\/.+@([\d\.]+).js$`)
	pkgVersion := map[string]string{}
	for pkg, url := range m.modules.Map {
		if !strings.HasPrefix(url, "vendor/") {
			continue
		}

		match := pkgReg.FindStringSubmatch(url)
		if len(match) != 2 {
			continue
		}

		pkgVersion[pkg] = match[1]
	}

	if len(pkgVersion) == 0 {
		fmt.Println("[info] no packages to check for vulnerabilities.")
		return nil
	}

	vulnerabilities, err := m.auditor.Audit(ctx, pkgVersion)
	if err != nil {
		return err
	}

	if len(vulnerabilities) == 0 {
		fmt.Println("[info] No vulnerable packages found.")
		return nil
	}

	var sortedPackages []string
	for pkg := range vulnerabilities {
		sortedPackages = append(sortedPackages, pkg)
	}

	slices.Sort(sortedPackages)

	fmt.Println("[info] Report results:")
	for _, pkg := range sortedPackages {
		fmt.Printf("\n%s@%s\n", pkg, pkgVersion[pkg])
		for _, vulnerability := range vulnerabilities[pkg] {
			fmt.Printf("  Severity            %q\n", vulnerability["severity"])
			fmt.Printf("  Description         %q\n", vulnerability["description"])
			fmt.Printf("  Vulnerable versions %q\n", vulnerability["vulnerable_versions"])
		}
	}

	return nil
}

func (m *manager) OutdatedPackages(ctx context.Context) error {
	pkgReg := regexp.MustCompile(`\/.+@([\d\.]+).js$`)
	pkgVersion := map[string]string{}
	for pkg, url := range m.modules.Map {
		if !strings.HasPrefix(url, "vendor/") {
			continue
		}

		match := pkgReg.FindStringSubmatch(url)
		if len(match) != 2 {
			continue
		}

		pkgVersion[pkg] = match[1]
	}

	outdated, err := m.auditor.Outdated(ctx, pkgVersion)
	if err != nil {
		return err
	}

	if len(outdated) == 0 {
		fmt.Println("[info] All packages are up to date.")
	}

	var maxTrailSpace int
	var packages []string
	for k := range outdated {
		packages = append(packages, k)
		maxTrailSpace = max(maxTrailSpace, len(k))
	}

	slices.Sort(packages)

	for _, k := range packages {
		trailingSpaces := strings.Repeat(" ", max(maxTrailSpace-len(k), 0))
		fmt.Printf("  %s%s pinned: %s, latest: %s\n", k, trailingSpaces, pkgVersion[k], outdated[k])
	}

	return nil
}
