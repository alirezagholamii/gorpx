package static

import "net/http"

func New(prefix, root string) http.Handler {
	fileServer := http.FileServer(http.Dir(root))

	return http.StripPrefix(prefix, fileServer)
}
