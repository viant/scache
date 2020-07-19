package scache

var noSuchKeyErr = &NoSuchKey{}

//NoSuchKey represents no such key error
type NoSuchKey struct{}

//Error returns key error
func (e NoSuchKey) Error() string {
	return "key not found"
}
