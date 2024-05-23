package simplesessions

// MockStore mocks the store for testing
type MockStore struct {
	err  error
	id   string
	data map[string]interface{}
}

func (s *MockStore) Create() (string, error) {
	return s.id, s.err
}

func (s *MockStore) Get(id, key string) (interface{}, error) {
	if s.id == "" {
		return nil, ErrInvalidSession
	}

	d, ok := s.data[key]
	if !ok {
		return nil, ErrFieldNotFound
	}
	return d, s.err
}

func (s *MockStore) GetMulti(id string, keys ...string) (values map[string]interface{}, err error) {
	if s.id == "" {
		return nil, ErrInvalidSession
	}

	out := make(map[string]interface{})
	for _, key := range keys {
		v, ok := s.data[key]
		if !ok {
			v = err
		}
		out[key] = v
	}

	return out, s.err
}

func (s *MockStore) GetAll(id string) (values map[string]interface{}, err error) {
	if s.id == "" {
		return nil, ErrInvalidSession
	}

	return s.data, s.err
}

func (s *MockStore) Set(cv, key string, value interface{}) error {
	if s.id == "" {
		return ErrInvalidSession
	}

	s.data[key] = value
	return s.err
}

func (s *MockStore) SetMulti(id string, data map[string]interface{}) error {
	if s.id == "" {
		return ErrInvalidSession
	}

	for k, v := range data {
		s.data[k] = v
	}
	return s.err
}

func (s *MockStore) Delete(id string, key ...string) error {
	if s.id == "" {
		return ErrInvalidSession
	}

	for _, k := range key {
		delete(s.data, k)
	}
	return s.err
}

func (s *MockStore) Clear(id string) error {
	if s.id == "" {
		return ErrInvalidSession
	}

	s.data = map[string]interface{}{}
	return s.err
}

func (s *MockStore) Int(inp interface{}, err error) (int, error) {
	return inp.(int), err
}

func (s *MockStore) Int64(inp interface{}, err error) (int64, error) {
	return inp.(int64), err
}

func (s *MockStore) UInt64(inp interface{}, err error) (uint64, error) {
	return inp.(uint64), err
}

func (s *MockStore) Float64(inp interface{}, err error) (float64, error) {
	return inp.(float64), err
}

func (s *MockStore) String(inp interface{}, err error) (string, error) {
	return inp.(string), err
}

func (s *MockStore) Bytes(inp interface{}, err error) ([]byte, error) {
	return inp.([]byte), err
}

func (s *MockStore) Bool(inp interface{}, err error) (bool, error) {
	return inp.(bool), err
}
