package storage

import (
	"encoding/binary"
	"testing"
)

func dataSetOne() ([]slot, [][]byte) {
	slot := []slot{
		{
			offset: 8192,
			length: 14,
		},
		{
			offset: 8178,
			length: 12,
		},
		{
			offset: 8166,
			length: 13,
		},
		{
			offset: 8153,
			length: 14,
		},
	}

	rawData := [][]byte{
		[]byte("1, Alice, Band"),
		[]byte("2, Ali, King"),
		[]byte("3, Max, Marcs"),
		[]byte("4, Rudy, Botha"),
	}

	return slot, rawData
}

func TestSlottedPageNewPage(t *testing.T) {
	// TESTING
	var expectedChecksum uint32 = 2675191381

	sp, err := NewSlottedPage(1)
	if err != nil {
		t.Fatal(err)
	}
	if sp == nil {
		t.Fatal("No page was created.")
	}

	checkChecksum := binary.LittleEndian.Uint32(sp.Data[OffsetChecksum:])
	if expectedChecksum != checkChecksum {
		t.Fatalf("error: checksum mismatch\nExpected Checksum: %v\t Checksum: %v\n", expectedChecksum, checkChecksum)
	}
}
