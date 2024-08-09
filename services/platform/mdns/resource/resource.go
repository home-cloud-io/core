package resource

type Action int64

const (
	Undefined Action = iota
	Added
	Deleted
	Updated
)

// Resource represents a resource to advertise over mDNS
type Resource struct {
	SourceType string
	Action     Action
	IPs        []string
	Name       string
	Namespace  string
}
