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

func (s *MockStore) IsValid(ss *Session, cv string) (isExist bool, err error) {
	return s.isValid, s.err
}

func (s *MockStore) Create(ss *Session) (cv string, err error) {
	return s.val.(string), s.err
}

func (s *MockStore) Get(ss *Session, cv, key string) (value interface{}, err error) {
	return s.val, s.err
}

func (s *MockStore) GetMulti(ss *Session, cv string, keys ...string) (values map[string]interface{}, err error) {
	vals := make(map[string]interface{})
	vals["val"] = s.val
	return vals, s.err
}

func (s *MockStore) GetAll(ss *Session, cv string) (values map[string]interface{}, err error) {
	vals := make(map[string]interface{})
	vals["val"] = s.val
	return vals, s.err
}

func (s *MockStore) Set(ss *Session, cv, key string, value interface{}) error {
	s.val = value
	return s.err
}

func (s *MockStore) Commit(ss *Session, cv string) error {
	s.isCommited = true
	return s.err
}

func (s *MockStore) Clear(ss *Session, cv string) error {
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
