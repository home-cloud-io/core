package strings

func Default(value string, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}
