package fields

// OptField is a field that may or may not exist.
// You can use it to represent optional fields in a struct.
// Its sort of like a pointer, but without the indirection.
// PATCH requests can use this for partial updates. If the field is not set,
// the field will not be updated. Using pointers for PATCH requests is not always
// ideal because you some of your fields might take nil as a valid value for update,
// so ignoring nil pointers would be incorrect.
type OptField[T any] struct {
	val    T
	exists bool
}

func (f OptField[T]) Get() (T, bool) {
	return f.val, f.exists
}

func (f *OptField[T]) Set(val T) {
	f.val = val
	f.exists = true
}

func (f *OptField[T]) Unset() {
	f.exists = false
}

func (f OptField[T]) IsSet() bool {
	return f.exists
}

func (f OptField[T]) IsUnset() bool {
	return !f.exists
}

func (f OptField[T]) OrElse(alt T) T {
	if f.exists {
		return f.val
	}
	return alt
}

func (f OptField[T]) OrElseGet(alt func() T) T {
	if f.exists {
		return f.val
	}
	return alt()
}
