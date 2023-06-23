// Copyright (C) 2021-2023 Nicola Murino
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, version 3.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package db

import (
	"errors"
	"path"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Metadater struct{}

func (m *Metadater) SetModificationTime(storageID, objectPath string, mTime int64) error {
	sess, cancel := getDefaultSession()
	defer cancel()

	err := executeTx(sess, func(tx *gorm.DB) error {
		folder := Folder{
			StorageID: storageID,
		}

		folderPath := path.Dir(objectPath)
		folder.Path = folderPath
		err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{
					Name: "path",
				},
				{
					Name: "storage_id",
				},
			},
			DoNothing: true,
		}).Create(&folder).Error
		if err != nil {
			return err
		}
		if folder.ID == 0 {
			folder = Folder{}
			err = tx.Where("path = ? AND storage_id = ?", folderPath, storageID).Select("id").First(&folder).Error
			if err != nil {
				return err
			}
		}
		file := File{
			Name:         path.Base(objectPath),
			LastModified: mTime,
			FolderID:     folder.ID,
		}
		return tx.Omit("Folder").Clauses(
			clause.OnConflict{
				Columns: []clause.Column{
					{
						Name: "name",
					},
					{
						Name: "folder_id",
					},
				},
				DoUpdates: clause.AssignmentColumns([]string{"last_modified"}),
			}).Create(&file).Error
	})

	return m.checkError(err)
}

func (m *Metadater) GetModificationTime(storageID, objectPath string) (int64, error) {
	var err error
	folder := Folder{}
	sess, cancel := getDefaultSession()
	defer cancel()

	err = sess.Where("path = ? AND storage_id = ?", path.Dir(objectPath), storageID).Select("id").First(&folder).Error
	if err != nil {
		return 0, m.checkError(err)
	}
	file := File{}
	err = sess.Where("name = ? AND folder_id = ?", path.Base(objectPath), folder.ID).Select("last_modified").First(&file).Error
	if err != nil {
		return 0, m.checkError(err)
	}
	return file.LastModified, nil
}

func (m *Metadater) GetModificationTimes(storageID, objectPath string) (map[string]int64, error) {
	sess, cancel := getSessionWithTimeout(defaultQueryTimeout * 4)
	defer cancel()

	result := make(map[string]int64)
	folder := Folder{}
	err := sess.Where("path = ? AND storage_id = ?", objectPath, storageID).Select("id").First(&folder).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return result, nil
		}
		return nil, m.checkError(err)
	}
	var files []File
	err = sess.Where("folder_id = ?", folder.ID).Select("name,last_modified").Find(&files).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return result, nil
		}
		return nil, m.checkError(err)
	}
	for idx := range files {
		result[files[idx].Name] = files[idx].LastModified
	}
	return result, nil
}

func (m *Metadater) RemoveMetadata(storageID, objectPath string) error {
	sess, cancel := getDefaultSession()
	defer cancel()

	folder := Folder{}
	err := sess.Where("path = ? AND storage_id = ?", path.Dir(objectPath), storageID).Select("id").First(&folder).Error
	if err != nil {
		return m.checkError(err)
	}
	sess = sess.Where("name = ? AND folder_id = ?", path.Base(objectPath), folder.ID).Delete(&File{})
	err = checkRowsAffected(sess)
	return m.checkError(err)
}

func (m *Metadater) GetFolders(storageID string, limit int, from string) ([]string, error) {
	var folders []Folder

	sess, cancel := getDefaultSession()
	defer cancel()

	if limit > 0 {
		sess = sess.Limit(limit)
	}
	if from != "" {
		sess = sess.Where("path > ?", from)
	}
	if storageID != "" {
		sess = sess.Where("storage_id = ?", storageID)
	}

	sess = sess.Order("path ASC")
	err := sess.Select("path").Find(&folders).Error
	if err != nil {
		return nil, m.checkError(err)
	}

	results := make([]string, 0, len(folders))
	for idx := range folders {
		results = append(results, folders[idx].Path)
	}

	return results, nil
}

func (m *Metadater) checkError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		st := status.New(codes.NotFound, err.Error())
		return st.Err()
	}
	return err
}
