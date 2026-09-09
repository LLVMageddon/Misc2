package storage

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func getDm(t *testing.T) (*DiskManager, error) {
	path := filepath.Join(t.TempDir(), "test.db")
	dm, err := NewDiskManger(path)
	defer dm.Close()
	return dm, err
}

func TestDiskManagerCreate(t *testing.T) {

	path := filepath.Join(t.TempDir(), "test.db")
	dm, err := NewDiskManger(path)
	if err != nil {
		t.Fatalf("failed to create disk manager: %v", err)
	}

	defer dm.Close()

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("database file was not created: %v", err)
	}

}

func TestAllocateFirstPage(t *testing.T) {
	dm, err := getDm(t)
	if err != nil {
		t.Fatalf("failed to create disk manager: %v", err)
	}
	// defer dm.Close()

	pageId, err := dm.AllocatePage()
	if err != nil {
		t.Fatalf("AllocatePage failed: %v", err)
	}
	if pageId != 1 {
		t.Fatalf("expected first id 1, got %d", pageId)
	}
}

func TestAllocateMultiplePages(t *testing.T) {
	dm, _ := getDm(t)
	// defer dm.Close()

	for id := uint64(0); id < 10; id++ {
		pageId, err := dm.AllocatePage()
		if err != nil {
			t.Fatalf("AllocatePage failed: %v", err)
		}
		if pageId != id+1 {
			t.Fatalf("expected page id %d, got %d", id+1, pageId)
		}
	}
}

func TestWriteAndReadPage(t *testing.T) {
	dm, _ := getDm(t)
	// defer dm.Close()

	pageId, err := dm.AllocatePage()
	if err != nil {
		t.Fatal(err)
	}

	page, _ := NewSlottedPage(pageId)
	recordId, _ := page.AddRecord([]byte("record 1"))

	if err := dm.WritePage(pageId, &page.Data); err != nil {
		t.Fatalf("WritePage failed: %v", err)
	}

	var readData [PageSize]byte
	if err := dm.ReadPage(pageId, &readData); err != nil {
		t.Fatalf("ReadPage failed: %v", err)
	}

	var readPage SlottedPage
	copy(readPage.Data[:], readData[:])

	readRecord, _ := readPage.GetRecord(recordId)
	if !bytes.Equal([]byte("record 1"), readRecord) {
		t.Fatal("read page does not match written page")
	}
}

func TestWriteFullPage(t *testing.T) {
	dm, _ := getDm(t)

	pageId, err := dm.AllocatePage()
	if err != nil {
		t.Fatal(err)
	}

	page, _ := NewSlottedPage(pageId)
	maxRecord := bytes.Repeat([]byte("X"), PageSize-HeaderSize-SlotSize-1) // NOTE: I haven't decide if I want padding between slots and records
	recordId, err := page.AddRecord(maxRecord)
	if err != nil {
		t.Fatalf("addRecord failed: %v", err)
	}

	if err := dm.WritePage(pageId, &page.Data); err != nil {
		t.Fatalf("writePage failed: %v", err)
	}

	var readData [PageSize]byte
	if err := dm.ReadPage(pageId, &readData); err != nil {
		t.Fatalf("ReadPage failed: %v", err)
	}

	var readPage SlottedPage
	copy(readPage.Data[:], readData[:])

	readRecord, _ := readPage.GetRecord(recordId)
	if !bytes.Equal(maxRecord, readRecord) {
		t.Fatal("full page was not persitsed correctly")
	}
}

func TestWriteZeroPage(t *testing.T) {
	dm, _ := getDm(t)

	pageId, err := dm.AllocatePage()
	if err != nil {
		t.Fatal(err)
	}

	page, _ := NewSlottedPage(pageId)
	if err := dm.WritePage(pageId, &page.Data); err != nil {
		t.Fatalf("Write Page failed: %v", err)
	}

	var readData [PageSize]byte
	if err := dm.ReadPage(pageId, &readData); err != nil {
		t.Fatalf("ReadPage failed: %v", err)
	}

	var readPage SlottedPage

	copy(readPage.Data[:], readData[:])
	if !bytes.Equal(page.Data[:], readPage.Data[:]) {
		t.Fatal("zero page changed after round trip")
	}

}

func TestPagePersistsAfterReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	dm, err := NewDiskManger(path)
	if err != nil {
		t.Fatal(err)
	}

	pageId, err := dm.AllocatePage()
	if err != nil {
		t.Fatal(err)
	}

	page, _ := NewSlottedPage(pageId)
	recordId, _ := page.AddRecord([]byte("persistant data"))

	if err := dm.WritePage(pageId, &page.Data); err != nil {
		t.Fatal(err)
	}

	if err := dm.Close(); err != nil {
		t.Fatal(err)
	}

	dm, err = NewDiskManger(path)
	if err != nil {
		t.Fatal(err)
	}

	defer dm.Close()

	var readData [PageSize]byte
	if err := dm.ReadPage(pageId, &readData); err != nil {
		t.Fatal(err)
	}

	var readPage SlottedPage
	copy(readPage.Data[:], readData[:])

	readRecord, _ := readPage.GetRecord(recordId)

	if !bytes.Equal([]byte("persistant data"), readRecord) {
		t.Fatal("page did not survive reopen")
	}
}

func TestPagesAreIndependent(t *testing.T) {
	dm, _ := getDm(t)

	page1Id, err := dm.AllocatePage()
	if err != nil {
		t.Fatal(err)
	}
	page1, _ := NewSlottedPage(page1Id)
	record1Id, _ := page1.AddRecord([]byte("page1 record1"))
	if err := dm.WritePage(page1Id, &page1.Data); err != nil {
		t.Fatal(err)
	}

	page2Id, err := dm.AllocatePage()
	if err != nil {
		t.Fatal(err)
	}
	page2, _ := NewSlottedPage(page2Id)
	record2Id, _ := page2.AddRecord([]byte("page2 record1"))
	if err := dm.WritePage(page2Id, &page2.Data); err != nil {
		t.Fatal(err)
	}

	var readData1, readData2 [PageSize]byte

	if err := dm.ReadPage(page1Id, &readData1); err != nil {
		t.Fatal(err)
	}

	if err := dm.ReadPage(page2Id, &readData2); err != nil {
		t.Fatal(err)
	}

	var readPage1, readPage2 SlottedPage
	copy(readPage1.Data[:], readData1[:])
	copy(readPage2.Data[:], readData2[:])

	readRecord1, _ := readPage1.GetRecord(record1Id)
	readRecord2, _ := readPage2.GetRecord(record2Id)

	if !bytes.Equal([]byte("page1 record1"), readRecord1) {
		t.Fatal("page 1 was corrupted")
	}

	if !bytes.Equal([]byte("page2 record1"), readRecord2) {
		t.Fatal("page 2 was corrupted")
	}
}

func TestOverWritePage(t *testing.T) {
	dm, _ := getDm(t)
	record1 := []byte("record 1")
	newRecord1 := []byte("record 2 that over wrote record 1")
	pageId, err := dm.AllocatePage()
	if err != nil {
		t.Fatal(err)
	}

	page, _ := NewSlottedPage(pageId)
	recordId, _ := page.AddRecord(record1)

	if err := dm.WritePage(pageId, &page.Data); err != nil {
		t.Fatal(err)
	}

	var readData [PageSize]byte
	if err := dm.ReadPage(pageId, &readData); err != nil {
		t.Fatal(err)
	}

	copy(page.Data[:], readData[:])
	readRecord1, _ := page.GetRecord(recordId)
	if !bytes.Equal(record1, readRecord1) {
		t.Fatal("read back record failed")
	}

	page.RemoveRecord(recordId)
	page.AddRecord(newRecord1)
	if err := dm.WritePage(pageId, &page.Data); err != nil {
		t.Fatal(err)
	}

	if err := dm.ReadPage(pageId, &readData); err != nil {
		t.Fatal(err)
	}
	copy(page.Data[:], readData[:])
	readRecord2, _ := page.GetRecord(recordId)
	if !bytes.Equal(newRecord1, readRecord2) {
		t.Fatal("read back record failed")
	}
}

func TestResultUnallocatedPage(t *testing.T) {
	dm, _ := getDm(t)

	page, _ := NewSlottedPage(1)

	if err := dm.ReadPage(999, &page.Data); err == nil {
		t.Fatal("expected error when reading unallocated page")
	}
}

func TestWriteUnallocatedPage(t *testing.T) {
	dm, _ := getDm(t)
	page, _ := NewSlottedPage(1)

	if err := dm.WritePage(999, &page.Data); err == nil {
		t.Fatal("expecred error when writting unallocated page")
	}
}

func TestMaxPageId(t *testing.T) {
	dm, _ := getDm(t)
	pageId, _ := dm.AllocatePage()
	page, _ := NewSlottedPage(pageId)
	err := dm.ReadPage(^uint64(0), &page.Data)
	if err == nil {
		t.Fatal("expected error for invalud maximum page ID")
	}
}

func TestReadFromEmpty(t *testing.T) {
	dm, _ := getDm(t)
	page, _ := NewSlottedPage(0)
	if err := dm.ReadPage(0, &page.Data); err == nil {
		t.Fatal("expected error reading from empty database")
	}
}

func TestAllocateManyPages(t *testing.T) {
	dm, _ := getDm(t)

	const pageCount = 1000

	for i := uint64(0); i < pageCount; i++ {
		pageID, err := dm.AllocatePage()
		if err != nil {
			t.Fatalf("failed allocating page %d: %v", i, err)
		}

		if pageID != i {
			t.Fatalf("expected page ID %d, got %d", i, pageID)
		}
	}
}

func TestReadWriteManyPages(t *testing.T) {
	dm, _ := getDm(t)

	const pageCount = 100

	for i := uint64(0); i < pageCount; i++ {
		pageId, err := dm.AllocatePage()
		if err != nil {
			t.Fatal(err)
		}

		page, _ := NewSlottedPage(pageId)

		if err := dm.WritePage(pageId, &page.Data); err != nil {
			t.Fatal(err)
		}
	}

	for i := uint64(0); i < pageCount; i++ {
		page, _ := NewSlottedPage(0)

		if err := dm.ReadPage(i+1, &page.Data); err != nil {
			t.Fatal(err)
		}

		got := binary.LittleEndian.Uint64(page.Data[OffsetPageId:])

		if got != i+1 {
			t.Fatalf("page %d contains data for page %d", i+1, got)
		}
	}
}

func TestReadAfterClose(t *testing.T) {
	dm, _ := getDm(t)

	pageID, err := dm.AllocatePage()
	if err != nil {
		t.Fatal(err)
	}

	if err := dm.Close(); err != nil {
		t.Fatal(err)
	}

	var page [PageSize]byte

	if err := dm.ReadPage(pageID, &page); err == nil {
		t.Fatal("expected read after close to fail")
	}
}

func TestWriteAfterClose(t *testing.T) {
	dm, _ := getDm(t)

	pageID, err := dm.AllocatePage()
	if err != nil {
		t.Fatal(err)
	}

	if err := dm.Close(); err != nil {
		t.Fatal(err)
	}

	var page [PageSize]byte

	if err := dm.WritePage(pageID, &page); err == nil {
		t.Fatal("expected write after close to fail")
	}
}

func TestDoubleClose(t *testing.T) {
	dm, _ := getDm(t)

	if err := dm.Close(); err != nil {
		t.Fatalf("first Close failed: %v", err)
	}

	if err := dm.Close(); err != nil {
		t.Fatalf("second Close should be safe, got: %v", err)
	}
}

func TestFileSizeMatchesAllocatedPages(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")

	dm, err := getDm(t)
	if err != nil {
		t.Fatal(err)
	}
	defer dm.Close()

	const pageCount = 10

	for i := 0; i < pageCount; i++ {
		if _, err := dm.AllocatePage(); err != nil {
			t.Fatal(err)
		}
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	expectedSize := int64(pageCount * PageSize)

	if info.Size() != expectedSize {
		t.Fatalf(
			"expected database size %d, got %d",
			expectedSize,
			info.Size(),
		)
	}
}
