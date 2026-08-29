package collectionstore

import "testing"

func TestEncodeDecodeFolderName_RoundTrips(t *testing.T) {
	encoded := EncodeFolderName(10, "users")
	name, order := DecodeName(encoded, false)
	if name != "users" || order != 10 {
		t.Errorf("got name=%q order=%d, want name=users order=10", name, order)
	}
}

func TestEncodeDecodeRequestName_RoundTrips(t *testing.T) {
	encoded := EncodeRequestFileName(20, "get-user")
	if encoded != "020_get-user.json" {
		t.Errorf("encoded = %q, want 020_get-user.json", encoded)
	}
	name, order := DecodeName(encoded, true)
	if name != "get-user" || order != 20 {
		t.Errorf("got name=%q order=%d, want name=get-user order=20", name, order)
	}
}

func TestDecodeName_UnprefixedFileSortsLast(t *testing.T) {
	name, order := DecodeName("legacy.json", true)
	if name != "legacy" {
		t.Errorf("name = %q, want legacy", name)
	}
	if order != UnorderedRank {
		t.Errorf("order = %d, want UnorderedRank (%d)", order, UnorderedRank)
	}
}

func TestDecodeName_UnprefixedFolderSortsLast(t *testing.T) {
	name, order := DecodeName("legacy-folder", false)
	if name != "legacy-folder" {
		t.Errorf("name = %q, want legacy-folder", name)
	}
	if order != UnorderedRank {
		t.Errorf("order = %d, want UnorderedRank (%d)", order, UnorderedRank)
	}
}

func TestNextOrder_EmptyStartsAtTen(t *testing.T) {
	if got := NextOrder(nil); got != 10 {
		t.Errorf("got %d, want 10", got)
	}
}

func TestNextOrder_AddsTenPastMax(t *testing.T) {
	if got := NextOrder([]int{10, 30, 20}); got != 40 {
		t.Errorf("got %d, want 40", got)
	}
}
