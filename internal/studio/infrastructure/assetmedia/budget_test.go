package assetmedia

import (
	"bytes"
	"io"
	"testing"
)

func TestBudgetReaderSharesLimitAcrossReaders(t *testing.T) {
	remaining := int64(10)
	first, err := io.ReadAll(&budgetReader{reader: bytes.NewReader([]byte("1234567")), remaining: &remaining})
	if err != nil {
		t.Fatal(err)
	}
	second, err := io.ReadAll(&budgetReader{reader: bytes.NewReader([]byte("abcdefghij")), remaining: &remaining})
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != "1234567" || string(second) != "abc" || remaining != 0 {
		t.Fatalf("read %q and %q with %d bytes remaining", first, second, remaining)
	}
}
