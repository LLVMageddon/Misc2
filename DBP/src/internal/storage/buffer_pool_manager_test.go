package storage

import (
	"fmt"
	"path/filepath"
	"testing"
)

func TestBufferPoolManagerCreateAndNewPage(t *testing.T) {
	// TESTING:
	path := filepath.Join(t.TempDir(), "test.db")
	dm, _ := NewDiskManger(path)
	defer dm.Close()

	bpm, err := NewBufferPoolManager(dm)
	if err != nil{
		t.Fatalf("create buffer pool manager failed: %v", err)
	}

	page, err := bpm.NewPage()
	if err != nil{
		t.Fatalf("new page failed: %v", err)
	}

	fmt.Println(page.GetPage())


}
