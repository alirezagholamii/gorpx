package static

import "net/http"

func New(prefix, root string) http.Handler {
	fs := fileSystem{
		root: root,
	}

	return http.StripPrefix(
		prefix,
		http.FileServer(fs),
	)
}
