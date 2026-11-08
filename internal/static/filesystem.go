package static

import (
	"errors"
	"net/http"
	"os"
	"strings"
)

type fileSystem struct {
	root string
}

func (fs fileSystem) Open(name string) (http.File, error) {
	name = strings.TrimPrefix(name, "/")

	if name == "" {
		name = "."
	}

	file, err := os.OpenInRoot(fs.root, name)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, os.ErrNotExist
		}

		return nil, os.ErrPermission
	}

	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, err
	}

	if info.IsDir() {
		file.Close()
		return nil, os.ErrPermission
	}

	if !info.Mode().IsRegular() {
		file.Close()
		return nil, os.ErrPermission
	}

	return file, nil
}
