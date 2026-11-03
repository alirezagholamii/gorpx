package static

import (
	"net/http"
	"os"
)

type fileSystem struct {
	http.FileSystem
}

func (fs fileSystem) Open(name string) (http.File, error) {
	file, err := fs.FileSystem.Open(name)
	if err != nil {
		return nil, err
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

	return file, nil
}
