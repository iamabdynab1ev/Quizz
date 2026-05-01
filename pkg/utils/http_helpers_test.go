package utils

import (
	"net/url"
	"reflect"
	"testing"
)

func TestParseFilterFromQueryParsesFields(t *testing.T) {
	values := url.Values{}
	values.Set("fields", "fio, phone_number ,fio")

	filter := ParseFilterFromQuery(values)

	want := []string{"fio", "phone_number", "fio"}
	if !reflect.DeepEqual(filter.Fields, want) {
		t.Fatalf("expected fields %v, got %v", want, filter.Fields)
	}
}
