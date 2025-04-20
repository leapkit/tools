package importmap

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/spf13/pflag"
	"go.leapkit.dev/tools/importmap/internal/jspm"
	"go.leapkit.dev/tools/importmap/internal/npm"
)

var containerFolder string

func init() {
	pflag.StringVar(&containerFolder, "importmap.folder", "./internal/system/assets", "The folder where the packaged are stored")
}

func Process(ctx context.Context) error {
	pflag.Parse()

	args := pflag.Args()

	if len(args) == 0 {
		fmt.Println("Usage: importmap [flags] <command> [arguments]")
		fmt.Println()
		fmt.Println("Available commands:")
		fmt.Println("      pin         Pin new packages")
		fmt.Println("      unpin       Unpin existing packages")
		fmt.Println("      update      Update all outdated pinned packages")
		fmt.Println("      pristine    Re-download all pinned packages")
		fmt.Println("      json        Show the full importmap in JSON")
		fmt.Println("      packages    Print out packages with version numbers")
		fmt.Println("      outdated    Check for outdated packages")
		fmt.Println("      audit       Run a security audit")
		fmt.Println()
		fmt.Println("Available flags:")
		pflag.PrintDefaults()

		return nil
	}

	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	m := NewManager(containerFolder, jspm.Client(), npm.Client())

	switch args[0] {
	case "pin":
		if len(args) < 2 {
			fmt.Println("[info] importmap pin <package> [package...]")
			return nil
		}

		if err := m.Pin(ctx, args[1:]...); err != nil {
			return fmt.Errorf("error pinning packages: %w", err)
		}

		fmt.Println("[info] Packages pinned successfully")
	case "unpin":
		if len(args) < 2 {
			fmt.Println("[info] importmap unpin <package> [packages...]")
			return nil
		}

		if err := m.Unpin(ctx, args[1:]...); err != nil {
			return fmt.Errorf("error unpinning packages: %w", err)
		}

		fmt.Println("[info] Packages unpinned successfully")
	case "update":
		if err := m.Update(ctx); err != nil {
			return fmt.Errorf("error updating packages: %w", err)
		}

		fmt.Println("[info] Packages updated successfully")
	case "pristine":
		fmt.Println("[info] re-downloading pinned packages:")
		if err := m.Pristine(ctx); err != nil {
			return fmt.Errorf("error re-downloading packages: %w", err)
		}

		fmt.Println("[info] Packages downloaded successfully")
	case "json":
		fmt.Println("[info]", filepath.Join(containerFolder, "importmap.json"))
		fmt.Printf("\n%s\n", m.JSON())
	case "packages":
		fmt.Println("[info] Pinned packages:")
		m.Packages()
	case "outdated":
		fmt.Println("[info] outdated packages:")
		m.OutdatedPackages(ctx)
	case "audit":
		fmt.Println("[info] audit report:")
		if err := m.Audit(ctx); err != nil {
			return fmt.Errorf("error refreshing packages: %w", err)
		}
	default:
		fmt.Println("[warn] unknown command")
	}

	return nil
}
