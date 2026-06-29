package key

// Key is a value for use with context.WithValue. It's used as
// a pointer so it fits in an interface{} without allocation.
type Key struct {
	Name string
}

func (k *Key) String() string {
	return "context value " + k.Name
}
