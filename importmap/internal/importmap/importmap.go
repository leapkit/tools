package importmap

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

var (
	containerFolder string
	env             string
	provider        string
)

func init() {
	flag.StringVar(&containerFolder, "importmap.folder", "./internal/system/assets", "the folder where the packaged are stored")
	flag.StringVar(&env, "importmap.env", "production", "the environment where the packaged are stored")
	flag.StringVar(&provider, "importmap.provider", "", "the environment where the packaged are stored")
}

func Process(ctx context.Context) error {
	flag.Parse()

	args := os.Args

	if len(args) < 2 {
		fmt.Println("Usage: importmap <command>")

		return nil
	}

	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	m := newManager()

	switch args[1] {
	case "pin":
		if len(args) < 3 {
			fmt.Println("[info] importmap pin <package> [package...]")
			return nil
		}

		if err := m.Pin(ctx, args[2:]...); err != nil {
			fmt.Println("[error] error pinning packages:", err)
			return err
		}

		fmt.Println("[info] Packages pinned successfully")
	case "unpin":
		if len(args) < 3 {
			fmt.Println("[info] importmap unpin <package> [packages...]")
			return nil
		}

		if err := m.Unpin(ctx, args[2:]...); err != nil {
			fmt.Println("[error] error pinning packages:", err)
			return err
		}

		fmt.Println("[info] Packages unpinned successfully")
	case "update":
		if err := m.Update(ctx); err != nil {
			fmt.Println("[error] error updating packages:", err)
			return err
		}

		fmt.Println("[info] Packages updated successfully")
	case "pristine":
		fmt.Println("[info] re-downloading pinned packages:")
		if err := m.Refresh(ctx); err != nil {
			fmt.Println("[error] error refreshing packages:", err)
			return err
		}

		fmt.Println("[info] Packages downloaded successfully")
	case "json":
		fmt.Printf("[info] %s:\n%s\n",
			filepath.Join(containerFolder, "importmap.json"),
			m.modules.Bytes(),
		)
	case "packages":
		fmt.Println("[info] Pinned packages:")
		m.Packages()
	case "outdated":
		fmt.Println("[info] Outdated packages:")
		m.OutdatedPackages(ctx)

	default:
		if len(args) == 1 {
			fmt.Println("[warn] no command")
			return nil
		}

		fmt.Println("[warn] unknown command")
	}

	return nil
}
