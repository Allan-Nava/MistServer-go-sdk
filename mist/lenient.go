package mist_go

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
)

// decodeLenient decodes a JSON object into the struct v points to. When a
// member has an unexpected JSON type it decodes the remaining members one by
// one and leaves only the mismatched fields at their zero value; a value that
// is not an object at all leaves the whole struct zero. Syntax errors are
// still returned.
//
// MistServer stores stream and config members verbatim from API calls, and
// sends them in nearly every reply, so one odd value must not break decoding
// of the whole response.
func decodeLenient(b []byte, v any) error {
	err := json.Unmarshal(b, v)
	var typeErr *json.UnmarshalTypeError
	if err == nil || !errors.As(err, &typeErr) {
		return err
	}

	rv := reflect.ValueOf(v).Elem()
	rv.Set(reflect.Zero(rv.Type()))
	var members map[string]json.RawMessage
	if json.Unmarshal(b, &members) != nil {
		return nil // not an object: keep the zero value
	}
	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		if !f.IsExported() {
			continue
		}
		name := strings.Split(f.Tag.Get("json"), ",")[0]
		if name == "-" {
			continue
		}
		if name == "" {
			name = f.Name
		}
		raw, ok := members[name]
		if !ok {
			for k, r := range members { // encoding/json matches names case-insensitively
				if strings.EqualFold(k, name) {
					raw, ok = r, true
					break
				}
			}
		}
		if ok {
			_ = json.Unmarshal(raw, rv.Field(i).Addr().Interface())
		}
	}
	return nil
}

// UnmarshalJSON decodes a stream leniently; see decodeLenient.
func (s *Stream) UnmarshalJSON(b []byte) error {
	type plain Stream
	return decodeLenient(b, (*plain)(s))
}

// UnmarshalJSON decodes the server config leniently; see decodeLenient.
func (c *Config) UnmarshalJSON(b []byte) error {
	type plain Config
	return decodeLenient(b, (*plain)(c))
}
