package db

import (
	"fmt"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestGetSetModificationTime(t *testing.T) {
	m := Metadater{}
	storageID := "s3://my-bucket"
	path1 := "/user1/folder1/file1.txt"
	path2 := "/user1/folder1/file2.txt"
	_, err := m.GetModificationTime(storageID, path1)
	checkNotFoundError(t, err)
	mTime := getTimeAsMsSinceEpoch(time.Now())
	err = m.SetModificationTime(storageID, path1, mTime)
	assert.NoError(t, err)
	mTimeGet, err := m.GetModificationTime(storageID, path1)
	assert.NoError(t, err)
	assert.Equal(t, mTime, mTimeGet)
	mTime += 100
	err = m.SetModificationTime(storageID, path1, mTime)
	assert.NoError(t, err)
	mTimeGet, err = m.GetModificationTime(storageID, path1)
	assert.NoError(t, err)
	assert.Equal(t, mTime, mTimeGet)
	_, err = m.GetModificationTime(storageID, path2)
	checkNotFoundError(t, err)
	mTime += 100
	err = m.SetModificationTime(storageID, path2, mTime)
	assert.NoError(t, err)
	mTimeGet, err = m.GetModificationTime(storageID, path2)
	assert.NoError(t, err)
	assert.Equal(t, mTime, mTimeGet)
	mTimeGet, err = m.GetModificationTime(storageID, path1)
	assert.NoError(t, err)
	assert.Equal(t, mTime-100, mTimeGet)
	// cleanup
	err = m.RemoveMetadata(storageID, path1)
	assert.NoError(t, err)
	err = m.RemoveMetadata(storageID, path2)
	assert.NoError(t, err)
	err = m.RemoveMetadata(storageID, path1)
	checkNotFoundError(t, err)
	err = m.RemoveMetadata(storageID, path2)
	checkNotFoundError(t, err)
	_, err = m.GetModificationTime(storageID, path1)
	checkNotFoundError(t, err)
	_, err = m.GetModificationTime(storageID, path2)
	checkNotFoundError(t, err)
}

func TestGetModificationTimes(t *testing.T) {
	m := Metadater{}
	storageID := "gs://my-bucket"
	folder1 := "user1/folder1"
	folder2 := "user1/folder2"
	for counter := 0; counter < 2; counter++ {
		for i := 0; i < 10; i++ {
			err := m.SetModificationTime(storageID, path.Join(folder1, fmt.Sprintf("folder1_file%v.txt", i)),
				getTimeAsMsSinceEpoch(time.Now()))
			assert.NoError(t, err)
			err = m.SetModificationTime(storageID, path.Join(folder2, fmt.Sprintf("folder2_file%v.txt", i)),
				getTimeAsMsSinceEpoch(time.Now()))
			assert.NoError(t, err)
		}
	}
	result, err := m.GetModificationTimes(storageID, folder1)
	assert.NoError(t, err)
	assert.Len(t, result, 10)

	result, err = m.GetModificationTimes(storageID, folder2)
	assert.NoError(t, err)
	assert.Len(t, result, 10)

	// cleanup
	for i := 0; i < 10; i++ {
		err = m.RemoveMetadata(storageID, path.Join(folder1, fmt.Sprintf("folder1_file%v.txt", i)))
		assert.NoError(t, err)
		err = m.RemoveMetadata(storageID, path.Join(folder2, fmt.Sprintf("folder2_file%v.txt", i)))
		assert.NoError(t, err)
	}

	result, err = m.GetModificationTimes(storageID, folder1)
	assert.NoError(t, err)
	assert.Len(t, result, 0)

	result, err = m.GetModificationTimes(storageID, folder2)
	assert.NoError(t, err)
	assert.Len(t, result, 0)

	err = removeUnreferencedFolders()
	assert.NoError(t, err)
	folders, err := m.GetFolders(storageID, 0, "")
	assert.NoError(t, err)
	assert.Len(t, folders, 0)
}

func TestGetFolders(t *testing.T) {
	m := Metadater{}
	storageID1 := "gs://my-bucket"
	storageID2 := "azblob://my-bucket"

	for i := 0; i < 10; i++ {
		err := m.SetModificationTime(storageID1, fmt.Sprintf("/folder%v/file.txt", i), getTimeAsMsSinceEpoch(time.Now()))
		assert.NoError(t, err)
		err = m.SetModificationTime(storageID2, fmt.Sprintf("/folder%v/file.txt", i), getTimeAsMsSinceEpoch(time.Now()))
		assert.NoError(t, err)
	}

	for _, storageID := range []string{storageID1, storageID2} {
		folders, err := m.GetFolders(storageID, 5, "")
		assert.NoError(t, err)
		if assert.Len(t, folders, 5) {
			assert.Equal(t, "/folder0", folders[0])
			assert.Equal(t, "/folder4", folders[4])
			folders, err = m.GetFolders(storageID1, 5, folders[4])
			assert.NoError(t, err)
			if assert.Len(t, folders, 5) {
				assert.Equal(t, "/folder5", folders[0])
				assert.Equal(t, "/folder9", folders[4])
			}
		}
	}

	folders1, err := m.GetFolders(storageID1, 0, "")
	assert.NoError(t, err)
	folders2, err := m.GetFolders(storageID2, 0, "")
	assert.NoError(t, err)
	assert.Len(t, folders1, 10)
	assert.Len(t, folders2, 10)

	for i := 0; i < 10; i++ {
		err = m.RemoveMetadata(storageID1, fmt.Sprintf("/folder%v/file.txt", i))
		assert.NoError(t, err)
	}

	folders1, err = m.GetFolders(storageID1, 0, "")
	assert.NoError(t, err)
	folders2, err = m.GetFolders(storageID2, 0, "")
	assert.NoError(t, err)
	assert.Len(t, folders1, 10)
	assert.Len(t, folders2, 10)

	err = removeUnreferencedFolders()
	assert.NoError(t, err)
	folders1, err = m.GetFolders(storageID1, 0, "")
	assert.NoError(t, err)
	folders2, err = m.GetFolders(storageID2, 0, "")
	assert.NoError(t, err)
	assert.Len(t, folders1, 0)
	assert.Len(t, folders2, 10)

	for i := 0; i < 10; i++ {
		err = m.RemoveMetadata(storageID2, fmt.Sprintf("/folder%v/file.txt", i))
		assert.NoError(t, err)
	}
	err = removeUnreferencedFolders()
	assert.NoError(t, err)
	folders2, err = m.GetFolders(storageID2, 0, "")
	assert.NoError(t, err)
	assert.Len(t, folders2, 0)
}

func TestFolderNameUniqueConstraint(t *testing.T) {
	m := Metadater{}
	storageID := "gs://mybucket"

	var sb strings.Builder
	sb.WriteString("/long/folder/name/")
	for i := 0; i < 4096; i++ {
		sb.WriteString("s")
	}
	folderName := sb.String()

	for i := 0; i < 10; i++ {
		err := m.SetModificationTime(storageID, path.Join(folderName, fmt.Sprintf("file%v.txt", i)),
			getTimeAsMsSinceEpoch(time.Now()))
		assert.NoError(t, err)
	}
	folders, err := m.GetFolders(storageID, 0, "")
	assert.NoError(t, err)
	assert.Len(t, folders, 1)

	files, err := m.GetModificationTimes(storageID, folderName)
	assert.NoError(t, err)
	assert.Len(t, files, 10)

	for i := 0; i < 10; i++ {
		err = m.RemoveMetadata(storageID, path.Join(folderName, fmt.Sprintf("file%v.txt", i)))
		assert.NoError(t, err)
	}

	files, err = m.GetModificationTimes(storageID, folderName)
	assert.NoError(t, err)
	assert.Len(t, files, 0)

	err = removeUnreferencedFolders()
	assert.NoError(t, err)
	folders, err = m.GetFolders(storageID, 0, "")
	assert.NoError(t, err)
	assert.Len(t, folders, 0)
}

func checkNotFoundError(t *testing.T, err error) {
	s, ok := status.FromError(err)
	if assert.True(t, ok) {
		assert.Equal(t, codes.NotFound, s.Code())
	}
}
