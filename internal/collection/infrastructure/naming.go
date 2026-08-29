// Package collectionstore implements collection.Store against a directory tree, following
// the storage layout in the project's collections/ folder: readable,
// git-diffable, no hidden state. Manual ordering is embedded directly in the
// visible filename ("010_name") rather than a side-car manifest, since a
// dotfile would itself be a form of hidden state.
package collectionstore

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// UnorderedRank is the sort rank given to a file/folder with no numeric
// prefix, so legacy or hand-created entries sort after ordered ones instead
// of interleaving unpredictably.
const UnorderedRank = math.MaxInt32

// EncodeFolderName builds the on-disk directory name for a folder at the
// given manual-sort order.
func EncodeFolderName(order int, name string) string {
	return fmt.Sprintf("%03d_%s", order, name)
}

// EncodeRequestFileName builds the on-disk file name for a request at the
// given manual-sort order.
func EncodeRequestFileName(order int, name string) string {
	return fmt.Sprintf("%03d_%s.json", order, name)
}

// DecodeName splits a directory or file name into its display name and
// manual-sort order, reversing EncodeFolderName/EncodeRequestFileName. Names
// without a recognized "NNN_" prefix get UnorderedRank.
func DecodeName(fileName string, isRequest bool) (name string, order int) {
	if isRequest {
		fileName = strings.TrimSuffix(fileName, ".json")
	}
	prefix, rest, found := strings.Cut(fileName, "_")
	if !found {
		return fileName, UnorderedRank
	}
	n, err := strconv.Atoi(prefix)
	if err != nil {
		return fileName, UnorderedRank
	}
	return rest, n
}

// NextOrder returns the order value for a new sibling appended after the
// existing ones, leaving gaps for later reordering.
func NextOrder(existing []int) int {
	highest := 0
	for _, o := range existing {
		if o > highest {
			highest = o
		}
	}
	return highest + 10
}
