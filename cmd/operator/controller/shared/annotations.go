package shared

import "strconv"

func IsAnnotationTrue(annotations map[string]string, key string) bool {
	v, err := strconv.ParseBool(annotations[key])
	return err == nil && v
}