package queryfields

import "testing"

func TestNormalizeUserListFieldsAddsIDAndDeduplicates(t *testing.T) {
	fields, err := NormalizeUserListFields([]string{"fio", "phone_number", "fio"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{"id", "fio", "phone_number"}
	if len(fields) != len(want) {
		t.Fatalf("expected %d fields, got %d", len(want), len(fields))
	}

	for i := range want {
		if fields[i] != want[i] {
			t.Fatalf("expected field %q at index %d, got %q", want[i], i, fields[i])
		}
	}
}

func TestNormalizeUserListFieldsRejectsUnknownField(t *testing.T) {
	if _, err := NormalizeUserListFields([]string{"fio", "role_ids"}); err == nil {
		t.Fatal("expected error for unsupported field")
	}
}
