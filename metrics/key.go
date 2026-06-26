package metrics

// Key is a value for use with context.WithValue. It's used as
// a pointer so it fits in an interface{} without allocation.
type key struct {
	Name string
}

func (k *key) String() string {
	return "context value " + k.Name
}
