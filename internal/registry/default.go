package registry

// DefaultTools is the list of bundled tools.
var DefaultTools = []Tool{
	{Name: "env-vault", Repo: "anivaryam/env-vault", Binary: "env-vault", Description: "Encrypted .env file manager"},
	{Name: "tunnel", Repo: "anivaryam/tunnel", Binary: "tunnel", Description: "Expose local services through a public tunnel"},
	{Name: "merge-port", Repo: "anivaryam/merge-port", Binary: "merge-port", Description: "Local reverse proxy that merges multiple ports into one"},
	{Name: "proc-compose", Repo: "anivaryam/proc-compose", Binary: "proc-compose", Description: "Process runner and manager with daemon support"},
	{Name: "proxy-relay", Repo: "anivaryam/proxy-relay", Binary: "proxy-relay", Description: "Authenticated SOCKS5/HTTP proxy client"},
}
