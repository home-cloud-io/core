package services

type Action int64

const (
	Undefined Action = iota
	Added
	Deleted
	Updated
)

// Resource represents a resource to advertise over mDNS
type Resource struct {
	Action    Action
	Hostname  string
	Name      string
	Namespace string
}
