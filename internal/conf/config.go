package conf

const defaultHTTPAddr = "0.0.0.0:8000"

type Config struct {
	HTTPAddr string
}

func Load(getenv func(string) string) Config {
	addr := getenv("HTTP_ADDR")
	if addr == "" {
		addr = defaultHTTPAddr
	}
	return Config{HTTPAddr: addr}
}
