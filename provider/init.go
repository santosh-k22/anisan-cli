package provider

const CustomProviderExtension = ".lua"

func init() {
	// Builtin providers are removed in favor of Lua scrapers
	// Use generic.Configuration if we ever decide to add a Go-based provider back
}
