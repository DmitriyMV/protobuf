package protobuf //nolint:testpackage

func SetGenerators(gr *generatorRegistry) {
	generators = gr
}

func GetGenerators() *generatorRegistry { return generators } //nolint:revive

func NewInterfaceRegistry() *generatorRegistry { //nolint:revive
	return newInterfaceRegistry()
}
