package config

type Server struct {
	Address string
}

type Route struct {
	Path     string
	Upstream string
}

type Config struct {
	Server Server
	Routes []Route
}

func Default() Config {
	return Config{
		Server: Server{
			Address: ":8080",
		},
		Routes: []Route{
			{
				Path:     "/api/",
				Upstream: "http://localhost:3000",
			},
			{
				Path:     "/app/",
				Upstream: "http://localhost:4200",
			},
		},
	}
}
