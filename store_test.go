package simplesessions

// MockStore mocks the store for testing
type MockStore struct {
	isValid     bool
	cookieValue string
	err         error
	val         interface{}
	isCommited  bool
}

func (s *MockStore) reset() {
	s.isValid = false
	s.cookieValue = ""
	s.err = nil
	s.val = nil
	s.isCommited = false
}

func (s *MockStore) Create() (cv string, err error) {
	return s.val.(string), s.err
}

func (s *MockStore) Get(cv, key string) (value interface{}, err error) {
	return s.val, s.err
}

func (s *MockStore) GetMulti(cv string, keys ...string) (values map[string]interface{}, err error) {
	vals := make(map[string]interface{})
	vals["val"] = s.val
	return vals, s.err
}

func (s *MockStore) GetAll(cv string) (values map[string]interface{}, err error) {
	vals := make(map[string]interface{})
	vals["val"] = s.val
	return vals, s.err
}

func (s *MockStore) Set(cv, key string, value interface{}) error {
	s.val = value
	return s.err
}

func (s *MockStore) Commit(cv string) error {
	s.isCommited = true
	return s.err
}

func (s *MockStore) Delete(cv string, key string) error {
	s.val = nil
	return s.err
}

func (s *MockStore) Clear(cv string) error {
	s.val = nil
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
