package importmap

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func (m *manager) Pin(ctx context.Context, packages ...string) error {
	if len(packages) == 0 {
		return nil
	}

	for _, pkg := range packages {
		if err := m.pinPackage(ctx, pkg); err != nil {
			return err
		}
	}

	return m.write()
}

func (m *manager) pinPackage(ctx context.Context, pkg string) error {
	// Track files for rollback
	var downloadedFiles []string

	// 1. Call JSPM
	genResult, err := m.generator.Generate(ctx, pkg)
	if err != nil {
		return err
	}

	result, ok := genResult.(GeneratorResult)
	if !ok {
		rollbackFiles(downloadedFiles)
		return fmt.Errorf("generator returned unexpected type")
	}

	imports := result.GetImports()
	scopes := result.GetScopes()
	staticDeps := result.GetStaticDeps()

	// 2. Determine main package name and version from imports
	// There should be exactly one entry in imports for the package we're pinning
	var mainPkgName, mainVersion string
	for packageName, packageURL := range imports {
		// Unpin if already exists
		if m.modules.Map[packageName] != "" {
			if err := m.Unpin(ctx, packageName); err != nil {
				rollbackFiles(downloadedFiles)
				return err
			}
		}

		// Parse to get main package info
		pkgName, version, relativePath, err := parseJSPMURL(packageURL)
		if err != nil {
			rollbackFiles(downloadedFiles)
			return fmt.Errorf("failed to parse package URL %s: %w", packageURL, err)
		}

		mainPkgName = pkgName
		mainVersion = version

		// Add to imports with vendor prefix
		vendorPath := "vendor/" + pkgName + "@" + version + "/" + relativePath
		m.modules.Map[packageName] = vendorPath
	}

	// 3. Download ALL files from staticDeps (this includes main + all dependencies)
	downloadedFiles, err = downloadAllStaticDeps(ctx, mainPkgName, mainVersion, staticDeps, m.containerFolder)
	if err != nil {
		rollbackFiles(downloadedFiles)
		return fmt.Errorf("failed to download files: %w", err)
	}

	// 4. Add scopes to importmap
	addScopesToImportMap(&m.modules, mainPkgName, mainVersion, scopes)

	fmt.Printf("[info] pinned %s (%d files)\n", pkg, len(downloadedFiles))

	return nil
}

// parseJSPMURL parses a JSPM URL into package name, version, and relative path
// Input:  "https://ga.jspm.io/npm:@org/package@1.2.3/dist/index.js"
// Output: "@org/package", "1.2.3", "dist/index.js"
func parseJSPMURL(url string) (pkgName, version, relativePath string, err error) {
	// Remove "https://ga.jspm.io/npm:"
	rest := strings.TrimPrefix(url, "https://ga.jspm.io/npm:")
	if rest == url {
		return "", "", "", fmt.Errorf("not a JSPM URL: %s", url)
	}

	// Handle scoped packages (@org/pkg@version/path) vs regular (pkg@version/path)
	var pathAfterVersion string

	if strings.HasPrefix(rest, "@") {
		// Scoped package: @org/package@1.2.3/dist/index.js
		// Find second @ (version delimiter)
		idx := strings.Index(rest[1:], "@")
		if idx == -1 {
			return "", "", "", fmt.Errorf("invalid scoped package format: %s", rest)
		}
		idx += 1 // Adjust for skipping first char

		pkgName = rest[:idx]
		remainder := rest[idx+1:] // After second @

		// Split version from path
		slashIdx := strings.Index(remainder, "/")
		if slashIdx > 0 {
			version = remainder[:slashIdx]
			pathAfterVersion = remainder[slashIdx+1:]
		} else {
			version = remainder
			pathAfterVersion = ""
		}
	} else {
		// Regular package: pkg@4.5.6/index.js
		parts := strings.SplitN(rest, "@", 2)
		if len(parts) != 2 {
			return "", "", "", fmt.Errorf("invalid package format: %s", rest)
		}

		pkgName = parts[0]
		remainder := parts[1]

		// Split version from path
		slashIdx := strings.Index(remainder, "/")
		if slashIdx > 0 {
			version = remainder[:slashIdx]
			pathAfterVersion = remainder[slashIdx+1:]
		} else {
			version = remainder
			pathAfterVersion = ""
		}
	}

	// Handle case where URL ends with just package@version (no path)
	if pathAfterVersion == "" {
		pathAfterVersion = "index.js" // Default entry point
	}

	return pkgName, version, pathAfterVersion, nil
}

// downloadFile downloads a file from URL to destPath with JSPM header comment
func downloadFile(url, destPath string) error {
	// Create directory structure
	dir := filepath.Dir(destPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Download
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, url)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Prepend comment header
	header := fmt.Sprintf("/* Downloaded from %s */\n\n", url)
	fullContent := append([]byte(header), content...)

	return os.WriteFile(destPath, fullContent, 0644)
}

// downloadAllStaticDeps downloads all files from staticDeps
// Returns list of downloaded file paths
func downloadAllStaticDeps(
	ctx context.Context,
	mainPkgName, mainVersion string,
	staticDeps []string,
	containerFolder string,
) ([]string, error) {
	var downloadedFiles []string

	for _, fileURL := range staticDeps {
		// Parse URL to get package name, version, and path
		pkgName, version, relativePath, err := parseJSPMURL(fileURL)
		if err != nil {
			return downloadedFiles, fmt.Errorf("failed to parse URL %s: %w", fileURL, err)
		}

		// Determine file path based on whether it's the main package or a dependency
		var filePath string
		if pkgName == mainPkgName && version == mainVersion {
			// Main package file - download to package root
			filePath = filepath.Join(
				containerFolder,
				"vendor",
				pkgName+"@"+version,
				relativePath,
			)
		} else {
			// Dependency file - download to scopes/
			filePath = filepath.Join(
				containerFolder,
				"vendor",
				mainPkgName+"@"+mainVersion,
				"scopes",
				pkgName+"@"+version,
				relativePath,
			)
		}

		if err := downloadFile(fileURL, filePath); err != nil {
			return downloadedFiles, fmt.Errorf("failed to download %s: %w", fileURL, err)
		}
		downloadedFiles = append(downloadedFiles, filePath)
	}

	return downloadedFiles, nil
}

// addScopesToImportMap adds scope entries to the importmap structure
func addScopesToImportMap(
	modules *modules,
	pkgName, version string,
	allScopes map[string]map[string]string,
) {
	// Find JSPM scope
	var jspmScope map[string]string
	for key, scope := range allScopes {
		if strings.Contains(key, "ga.jspm.io") {
			jspmScope = scope
			break
		}
	}

	if jspmScope == nil {
		return
	}

	// Create scope key for this package
	scopeKey := "vendor/" + pkgName + "@" + version + "/"

	if modules.Scopes == nil {
		modules.Scopes = make(map[string]map[string]string)
	}
	if modules.Scopes[scopeKey] == nil {
		modules.Scopes[scopeKey] = make(map[string]string)
	}

	// Map each dependency
	for depName, depURL := range jspmScope {
		depPkgName, depVersion, depRelPath, err := parseJSPMURL(depURL)
		if err != nil {
			// Skip invalid URLs
			continue
		}

		scopeValue := "vendor/" + pkgName + "@" + version + "/scopes/" +
			depPkgName + "@" + depVersion + "/" + depRelPath

		modules.Scopes[scopeKey][depName] = scopeValue
	}
}

// rollbackFiles deletes downloaded files and empty parent directories
func rollbackFiles(files []string) {
	for _, file := range files {
		os.Remove(file)

		// Remove empty parent directories up to vendor/
		dir := filepath.Dir(file)
		for dir != "vendor" && dir != "." && dir != "/" {
			if err := os.Remove(dir); err != nil {
				break // Directory not empty or error, stop
			}
			dir = filepath.Dir(dir)
		}
	}
}

func (m *manager) write() error {
	// Clean empty scopes
	if m.modules.Scopes != nil && len(m.modules.Scopes) == 0 {
		m.modules.Scopes = nil
	}

	outputPath := filepath.Join(m.containerFolder, "importmap.json")
	if err := os.MkdirAll(filepath.Dir(outputPath), os.ModePerm); err != nil {
		return err
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	f.Write(m.modules.Bytes())

	return nil
}
