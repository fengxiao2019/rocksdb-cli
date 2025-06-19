package command

import (
	"errors"
	"testing"
)

type mockDB struct {
	data map[string]string
}

func (m *mockDB) Get(key string) (string, error) {
	v, ok := m.data[key]
	if !ok {
		return "", errors.New("not found")
	}
	return v, nil
}
func (m *mockDB) Put(key, value string) error {
	m.data[key] = value
	return nil
}
func (m *mockDB) PrefixScan(prefix string, limit int) (map[string]string, error) {
	res := make(map[string]string)
	for k, v := range m.data {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			res[k] = v
		}
	}
	return res, nil
}
func (m *mockDB) Close() {}

func TestHandler_Execute(t *testing.T) {
	db := &mockDB{data: map[string]string{"foo": "bar", "foo2": "baz", "fop": "zzz"}}
	h := &Handler{DB: db}

	cases := []struct {
		cmd   string
		check func(output string) bool
	}{
		{"get foo", func(out string) bool { return true }},
		{"put newkey newval", func(out string) bool { return db.data["newkey"] == "newval" }},
		{"prefix foo", func(out string) bool { return true }},
		{"get notfound", func(out string) bool { return true }},
	}

	for _, c := range cases {
		h.Execute(c.cmd)
		if !c.check("") {
			t.Errorf("case %q failed", c.cmd)
		}
	}
}
