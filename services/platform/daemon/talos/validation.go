package talos

type ValidationMode struct{}

func (ValidationMode) String() string {
	return ""
}

func (ValidationMode) RequiresInstall() bool {
	return false
}

func (ValidationMode) InContainer() bool {
	return false
}
