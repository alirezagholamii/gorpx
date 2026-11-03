package static

import "net/http"

func New(prefix, root string) http.Handler {
	fs := fileSystem{
		FileSystem: http.Dir(root),
	}

	fileServer := http.FileServer(fs)

	return http.StripPrefix(prefix, fileServer)
}
