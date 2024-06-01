package simplesessions

// Store represents store interface. This interface can be
// implemented to create various backend stores for session.
type Store interface {
	// Create creates new session in the store for the given session ID.
	Create(id string) (err error)

	// Get a value for the given key from session.
	Get(id, key string) (value interface{}, err error)

	// GetMulti gets a maps of multiple values for given keys from session.
	// If some fields are not found then return nil for that field.
	GetMulti(id string, keys ...string) (data map[string]interface{}, err error)

	// GetAll gets all key and value from session.
	GetAll(id string) (data map[string]interface{}, err error)

	// Set sets an value for a field in session.
	Set(id, key string, value interface{}) error

	// Set takes a map of kv pair and set the field in store.
	SetMulti(id string, data map[string]interface{}) error

	// Delete a given list of keys from session.
	Delete(id string, key ...string) error

	// Clear empties the session but doesn't delete it.
	Clear(id string) error

	// Destroy deletes the entire session.
	Destroy(id string) error

	// Helper method for typecasting/asserting.
	// Supposed to be used as a chain.
	// For example: sess.Int(sess.Get("id", "key"))
	// Take `error` and returns that if its not nil.
	// Take `interface{}` value and type assert or convert.
	// If its nil then return ErrNil.
	// If it can't type asserted/converted then return ErrAssertType.
	Int(interface{}, error) (int, error)
	Int64(interface{}, error) (int64, error)
	UInt64(interface{}, error) (uint64, error)
	Float64(interface{}, error) (float64, error)
	String(interface{}, error) (string, error)
	Bytes(interface{}, error) ([]byte, error)
	Bool(interface{}, error) (bool, error)
}
