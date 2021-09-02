package timeout

// ErrTimeout is returned when a read operation took longer than the specifide duration
type ErrTimeout struct{}

func (err ErrTimeout) Error() string {
	return "Operation timed out"
}
