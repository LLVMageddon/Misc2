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
		recordId, err := sp.AddRecord(rawData[x])
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		if recordId != uint16(x+1) {
			t.Fatalf("error: adding record that should have a record id of %v", x+1)
		}
	}

	page, err := sp.GetPage()
	if err != nil {
		t.Fatal("error: deserializing the slotted page")
	}

	for x := 0; x < int(page.header.slotCount); x++ {
		if !bytes.Equal(page.data.data[x], rawData[x]) {
			t.Fatalf("error: adding record deserializing record %v", x+1)
		}
	}

}

// func TestAddZeroLengthRecord(t *testing.T) {
// 	// PASSED
// 	sp, _ := NewSlottedPage(1)
// 	slotId, err := sp.addRecord([]byte{})
// 	if err != nil {
// 		t.Fatalf("error: adding zero lenght should be allowed or handled cleanly, %v", err)
// 	}
// 	if slotId < 1 { //NOTE: slotId's start from 1
// 		t.Errorf("error: invalid slot ID returned: %d", slotId)
// 	}
// }

func TestReuseDeletedSlot(t *testing.T) {
	// PASSED
	sp, _ := NewSlottedPage(1)
	r1, _ := sp.AddRecord([]byte("record 1"))
	// r2, _ := sp.addRecord([]byte("record 2"))
	sp.AddRecord([]byte("record 2"))

	deleted, err := sp.RemoveRecord(r1)

	if !deleted {
		t.Fatalf("error: failed to delete: %v", err)
	}

	newSlotId, err := sp.AddRecord([]byte("record 3"))

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

	r1, _ := sp.AddRecord([]byte("Slot 0"))
	sp.AddRecord([]byte("Slot 1"))
	r3, _ := sp.AddRecord([]byte("Slot 2"))

	sp.RemoveRecord(r1)
	sp.RemoveRecord(r3)

	rr1, _ := sp.AddRecord([]byte("Reused 0"))
	rr2, _ := sp.AddRecord([]byte("Reused 2"))

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

	d1, _ := sp.AddRecord(make([]byte, 4123))
	// d2, _ := sp.addRecord(make([]byte, 4010))
	sp.AddRecord(make([]byte, 4010))

	deleted, err := sp.RemoveRecord(d1)
	if !deleted {
		t.Fatal(err)
	}

	d3, err := sp.AddRecord(make([]byte, 3500))

	if err != nil {
		t.Fatalf("expected add to succeed after automatic compactions, got: %v", err)
	}

	pp, _ := sp.GetPage()
	fmt.Print(pp)

	if d3 != d1 {
		t.Fatalf("expected id %v, got: %v", d1, d3)
	}

	record, err := sp.GetRecord(d3)
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
		_, err = sp.AddRecord(record)
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
	sp.AddRecord(make([]byte, 4080))
	sp.AddRecord(make([]byte, 4066))
	_, err := sp.AddRecord([]byte("Overflow"))

	if err == nil {
		t.Error("expected page full error when add to completely full page")
	}
}

func TestRecordToLarge(t *testing.T) {
	// PASSED
	sp, _ := NewSlottedPage(1)
	_, err := sp.AddRecord(make([]byte, (PageSize-HeaderSize)+5))
	// _, err := sp.addRecord(make([]byte, 0))
	if err == nil {
		t.Error("expected error for oversized recored, got nil")
	}
}

func TestAddRecordAndGetRecord(t *testing.T) {
	// PASSED
	sp, _ := NewSlottedPage(1)

	record1 := []byte("hello")
	rID, err := sp.AddRecord(record1)
	if err != nil {
		t.Fatalf("addRecord failed: %v", err)
	}

	if rID != 1 {
		t.Fatalf("expected record ID 1, got %d", rID)
	}

	got, err := sp.GetRecord(rID)
	if err != nil {
		t.Fatalf("getRecord failed: %v", err)
	}

	if !bytes.Equal(got, record1) {
		t.Fatalf("expected %q, got %q", record1, got)
	}

}

func TestAddRecordReusesDeletedSlot(t *testing.T) {
	// PASSED
	sp, _ := NewSlottedPage(1)

	id1, _ := sp.AddRecord([]byte("record 1"))
	id2, _ := sp.AddRecord([]byte("record 2"))

	if id1 != 1 || id2 != 2 {
		t.Fatalf("unexpected IDs: %d, %d", id1, id2)
	}

	if _, err := sp.RemoveRecord(id1); err != nil {
		t.Fatalf("rremoveRecord failed: %v", err)
	}

	id3, err := sp.AddRecord([]byte("record 3"))
	if err != nil {
		t.Fatalf("addRecord failed: %v", err)
	}
	if id3 != 1 {
		t.Fatalf("expected reused record ID 1, got %d", id3)
	}

	got, err := sp.GetRecord(1)
	if err != nil {
		t.Fatalf("getRecord failed: %v", err)
	}
	if !bytes.Equal(got, []byte("record 3")) {
		t.Fatalf("expected record 3, got %q", got)
	}
}

func TestCompactPagePreservesRecordID(t *testing.T) {
	// PASSED
	sp, _ := NewSlottedPage(1)

	id1, _ := sp.AddRecord([]byte("record 1"))
	id2, _ := sp.AddRecord([]byte("record 2"))
	id3, _ := sp.AddRecord([]byte("record 3"))

	if _, err := sp.RemoveRecord(id2); err != nil {
		t.Fatalf("removeRecord failed %v", err)
	}

	if err := sp.compactPage(); err != nil {
		t.Fatalf("compactPage failed: %v", err)
	}

	got1, err := sp.GetRecord(id1)
	if err != nil {
		t.Fatalf("getRecord(1) failed: %v", err)
	}

	got3, err := sp.GetRecord(id3)
	if err != nil {
		t.Fatalf("getRecord(3) failed: %v", err)
	}

	if !bytes.Equal(got1, []byte("record 1")) {
		t.Fatalf("record 1 changed: %q", got1)
	}

	if !bytes.Equal(got3, []byte("record 3")) {
		t.Fatalf("record 3 changed: %q", got3)
	}

	if _, err := sp.GetRecord(2); err == nil {
		t.Fatalf("expected record 2 to remain deleted")
	}

}

func TestAddRecordCompactThenReuseDeleteSlot(t *testing.T) {
	// PASSED
	sp, _ := NewSlottedPage(1)

	largeRecord := bytes.Repeat([]byte("A"), 2000)

	id1, err := sp.AddRecord(largeRecord)
	if err != nil {
		t.Fatalf("add record 1: %v", err)
	}

	id2, err := sp.AddRecord(largeRecord)
	if err != nil {
		t.Fatalf("add record 2: %v", err)
	}

	id3, err := sp.AddRecord(largeRecord)
	if err != nil {
		t.Fatalf("add record 3: %v", err)
	}

	if id1 != 1 || id2 != 2 || id3 != 3 {
		t.Fatalf("unexpected ID: %d %d %d", id1, id2, id3)
	}

	if _, err := sp.RemoveRecord(id2); err != nil {
		t.Fatalf("remove record 2: %v", err)
	}

	id4, err := sp.AddRecord(largeRecord)
	if err != nil {
		t.Fatalf("add record 4: %v", err)
	}
	if id4 != 2 {
		t.Fatalf("expeected record ID 2 to be reuesed after compaction, got %d", id4)
	}

	got, err := sp.GetRecord(2)
	if err != nil {
		t.Fatalf("getRecord(2): %v", err)
	}
	if !bytes.Equal(got, largeRecord) {
		t.Fatalf("reused slot contains incorrect record")
	}

}

func TestGetRecordZeroId(t *testing.T) {
	// PASSED
	sp, _ := NewSlottedPage(1)

	if _, err := sp.GetRecord(0); err == nil {
		t.Fatalf("expected error for record ID 0")
	}
}

func TestRemoveRecorcZeroID(t *testing.T) {
	// PASSED
	sp, _ := NewSlottedPage(1)

	if _, err := sp.RemoveRecord(0); err == nil {
		t.Fatalf("expected error fo record ID 0")
	}
}

func TestGetRecordOutOfBounds(t *testing.T) {
	// PASSED
	sp, _ := NewSlottedPage(1)
	if _, err := sp.GetRecord(999); err == nil {
		t.Fatalf("expected error for non existent record")
	}
}

func TestRemoveRecordTwice(t *testing.T) {
	// PASSED
	sp, _ := NewSlottedPage(1)
	id, err := sp.AddRecord([]byte("FRPS"))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := sp.RemoveRecord(id); err != nil {
		t.Fatal(err)
	}

	if _, err := sp.RemoveRecord(id); err == nil {
		t.Fatal("expected second remove to fail")
	}

}

func TestEmptyRecord(t *testing.T) {
	// PASSED
	sp, _ := NewSlottedPage(0)
	_, err := sp.AddRecord([]byte{})
	if err == nil {
		t.Fatal("expected empty record to be rejected")
	}
}

func TestMaxRecord(t *testing.T) {
	// PASSED
	sp, _ := NewSlottedPage(5)
	record := bytes.Repeat([]byte("X"), PageSize-HeaderSize-SlotSize-1) // NOTE: I haven't decide if I want padding between slots and records
	id, err := sp.AddRecord(record)
	if err != nil {
		t.Fatalf("maximum record failed: %v", err)
	}
	got, err := sp.GetRecord(id)
	if err != nil {
		t.Fatalf("getRecord failed: %v", err)
	}
	if !bytes.Equal(got, record) {
		t.Fatal("maximum record was coruupted")
	}
}

func TestREcordTooLarge(t *testing.T) {
	// PASSED
	sp, _ := NewSlottedPage(1)
	record := bytes.Repeat([]byte("S"), PageSize-HeaderSize-SlotSize+1)
	if _, err := sp.AddRecord(record); err == nil {
		t.Fatal("expected oversized record to fail")
	}
}

func TestManyDeltedSlots(t *testing.T) {
	// PASSED
	sp, _ := NewSlottedPage(1)
	ids := make([]uint16, 0, 20)
	for i := 0; i < 20; i++ {
		id, err := sp.AddRecord([]byte("record"))
		if err != nil {
			t.Fatalf("add record %d: %v", i, err)
		}

		ids = append(ids, id)
	}

	for i := 0; i < len(ids); i += 2 {
		if _, err := sp.RemoveRecord(ids[i]); err != nil {
			t.Fatalf("remove record %d: %v", ids[i], err)
		}
	}
}
