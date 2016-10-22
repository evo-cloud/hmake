package server

var (
	_modules []Module
)

// RegisterModule registers a module
func RegisterModule(m Module) {
	_modules = append(_modules, m)
}

// Modules returns all registered modules
func Modules() []Module {
	return _modules
}
