package simplesessions

// MockStore mocks the store for testing
type MockStore struct {
	err  error
	id   string
	data map[string]interface{}
	temp map[string]interface{}
}

func (s *MockStore) Create() (cv string, err error) {
	return s.id, s.err
}

func (s *MockStore) Get(cv, key string) (value interface{}, err error) {
	if s.id == "" {
		return nil, ErrInvalidSession
	}

	d, ok := s.data[key]
	if !ok {
		return nil, ErrFieldNotFound
	}

	return d, s.err
}

func (s *MockStore) GetMulti(cv string, keys ...string) (values map[string]interface{}, err error) {
	if s.id == "" {
		return nil, ErrInvalidSession
	}

	out := make(map[string]interface{})
	for _, key := range keys {
		v, err := s.Get(cv, key)
		if err != nil {
			if err == ErrFieldNotFound {
				v = nil
			} else {
				return nil, err
			}
		}
		out[key] = v
	}

	return out, s.err
}

func (s *MockStore) GetAll(cv string) (values map[string]interface{}, err error) {
	if s.id == "" {
		return nil, ErrInvalidSession
	}

	return s.data, s.err
}

func (s *MockStore) Set(cv, key string, value interface{}) error {
	if s.id == "" {
		return ErrInvalidSession
	}

	s.temp[key] = value
	return s.err
}

func (s *MockStore) Commit(cv string) error {
	if s.id == "" {
		return ErrInvalidSession
	}

	for key, val := range s.temp {
		s.data[key] = val
	}
	s.temp = map[string]interface{}{}

	return s.err
}

func (s *MockStore) Delete(cv string, key string) error {
	if s.id == "" {
		return ErrInvalidSession
	}
	s.temp = nil
	delete(s.data, key)
	delete(s.temp, key)
	return s.err
}

func (s *MockStore) Clear(cv string) error {
	if s.id == "" {
		return ErrInvalidSession
	}
	s.data = map[string]interface{}{}
	s.temp = map[string]interface{}{}
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
