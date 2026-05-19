package importmap

import (
	"os"
	"path/filepath"
)

// save writes the current importmap state to importmap.json
// in the container folder.
func (m *manager) save() error {
	outputPath := filepath.Join(m.containerFolder, "importmap.json")
	if err := os.MkdirAll(filepath.Dir(outputPath), os.ModePerm); err != nil {
		return err
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(m.modules.Bytes())
	return err
}
