package storage

import (
	"bytes"
	"encoding/binary"
	"fmt"
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
	// PASSED
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

func TestSlottedPageAddRecord(t *testing.T) {
	// PASSED
	sp, _ := NewSlottedPage(1)
	slots, rawData := dataSetOne()

	for x := 0; x < len(slots); x++ {
		recordId, err := sp.addRecord(rawData[x])
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		if recordId != uint16(x+1) {
			t.Fatalf("error: adding record that should have a record id of %v", x+1)
		}
	}

	page, err := sp.getPage()
	if err != nil {
		t.Fatal("error: deserializing the slotted page")
	}

	for x := 0; x < int(page.header.slotCount); x++ {
		if !bytes.Equal(page.data.data[x], rawData[x]) {
			t.Fatalf("error: adding record deserializing record %v", x+1)
		}
	}

}

func TestAddZeroLengthRecord(t *testing.T) {
	// PASSED
	sp, _ := NewSlottedPage(1)
	slotId, err := sp.addRecord([]byte{})
	if err != nil {
		t.Fatalf("error: adding zero lenght should be allowed or handled cleanly, %v", err)
	}
	if slotId < 1 { //NOTE: slotId's start from 1
		t.Errorf("error: invalid slot ID returned: %d", slotId)
	}
}

func TestReuseDeletedSlot(t *testing.T) {
	// PASSED
	sp, _ := NewSlottedPage(1)
	r1, _ := sp.addRecord([]byte("record 1"))
	// r2, _ := sp.addRecord([]byte("record 2"))
	sp.addRecord([]byte("record 2"))

	deleted, err := sp.removeRecord(r1)

	if !deleted {
		t.Fatalf("error: failed to delete: %v", err)
	}

	newSlotId, err := sp.addRecord([]byte("record 3"))

	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if newSlotId != r1 {
		t.Errorf("error: slot ID to be reused (%d), got %d", r1, newSlotId)
	}

}

func TestMultipleReuse(t *testing.T) {
	// PASSED
	sp, _ := NewSlottedPage(1)

	r1, _ := sp.addRecord([]byte("Slot 0"))
	sp.addRecord([]byte("Slot 1"))
	r3, _ := sp.addRecord([]byte("Slot 2"))

	sp.removeRecord(r1)
	sp.removeRecord(r3)

	rr1, _ := sp.addRecord([]byte("Reused 0"))
	rr2, _ := sp.addRecord([]byte("Reused 2"))

	if rr1 != r1 && rr1 != r3 {
		t.Errorf("expected rr1 to reuse slot %d or %d, got %d", r1, r3, rr1)
	}
	if rr2 != r1 && rr2 != r3 {
		t.Errorf("expected rr2 to reuse slot %d or %d, got %d", r1, r3, rr2)
	}

}

func TestCompactOnAdd(t *testing.T) {
	//FIXME: Logic is wrong
	// PASSED: wrong logic
	//FAILED
	sp, _ := NewSlottedPage(1)

	d1, _ := sp.addRecord(make([]byte, 4000))
	d2, _ := sp.addRecord(make([]byte, 4000))

	deleted, err := sp.removeRecord(d1)
	if !deleted {
		t.Fatal(err)
	}

	d3, err := sp.addRecord(make([]byte, 3500))

	if err != nil {
		t.Fatalf("expected add to succeed after automatic compactions, got: %v", err)
	}

	pp, _ := sp.getPage()
	fmt.Print(pp)

	if d3 != d2 {
		t.Fatalf("expected id %v, got: %v", d2, d3)
	}

	record, err := sp.getRecord(d3)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(make([]byte, 3500), record) {
		t.Fatalf("error: CompactOnAdd failed record is not equal")
	}

}

func TestFillPage(t *testing.T) {
	// PASSED
	sp, _ := NewSlottedPage(1)
	var err error
	var count int
	record := []byte("a")

	for {
		_, err = sp.addRecord(record)
		if err != nil {
			t.Logf("%v", err)
			break
		}

		count++
	}

	if count == 0 {
		t.Fatal("error: failed to add any tiny records")
	}

	if count != (PageSize-HeaderSize)/(SlotSize+len(record))-1 {
		t.Fatal("error: failed to add any tiny records")
	}
	t.Logf("Successfully filled page slot directory with %d tiny records before space exhausted", count)

}

func TestPageFullNoCompactionPossible(t *testing.T) {
	// PASSED
	sp, _ := NewSlottedPage(1)
	sp.addRecord(make([]byte, 4080))
	sp.addRecord(make([]byte, 4066))
	_, err := sp.addRecord([]byte("Overflow"))

	if err == nil {
		t.Error("expected page full error when add to completely full page")
	}
}

func TestRecordToLarge(t *testing.T) {
	// PASSED
	sp, _ := NewSlottedPage(1)
	_, err := sp.addRecord(make([]byte, (PageSize-HeaderSize)+5))
	// _, err := sp.addRecord(make([]byte, 0))
	if err == nil {
		t.Error("expected error for oversized recored, got nil")
	}
}
