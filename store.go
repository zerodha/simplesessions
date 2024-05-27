package simplesessions

// Store represents store interface. This interface can be
// implemented to create various backend stores for session.
type Store interface {
	// Create creates new session in the store and returns the session ID.
	Create(id string) (err error)

	// Get gets a value for given key from session.
	Get(id, key string) (value interface{}, err error)

	// GetMulti gets a maps of multiple values for given keys.
	// If some fields are not found then return ErrFieldNotFound for that field.
	GetMulti(id string, keys ...string) (data map[string]interface{}, err error)

	// GetAll gets all key and value from session.
	GetAll(id string) (data map[string]interface{}, err error)

	// Set sets an value for a field in session.
	Set(id, key string, value interface{}) error

	// Set takes a map of kv pair and set the field in store.
	SetMulti(id string, data map[string]interface{}) error

	// Delete a given list of keys from session.
	Delete(id string, key ...string) error

	// Clear clears the entire session.
	Clear(id string) error

	// Helper method for typecasting/asserting.
	Int(interface{}, error) (int, error)
	Int64(interface{}, error) (int64, error)
	UInt64(interface{}, error) (uint64, error)
	Float64(interface{}, error) (float64, error)
	String(interface{}, error) (string, error)
	Bytes(interface{}, error) ([]byte, error)
	Bool(interface{}, error) (bool, error)
}
