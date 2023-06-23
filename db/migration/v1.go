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

package migration

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

const (
	mignationV1ID = "1"
)

type folderV1 struct {
	ID        int64  `gorm:"primarykey"`
	Path      string `gorm:"type:text;not null;index:idx_folder_path;index:idx_unique_folder_path_storage_id,unique"`
	StorageID string `gorm:"size:512;not null;index:idx_folder_storage_id;index:idx_unique_folder_path_storage_id,unique"`
}

func (*folderV1) TableName() string {
	return "metadata_folders"
}

type fileV1 struct {
	ID           int64    `gorm:"primarykey"`
	Name         string   `gorm:"size:512;not null;index:idx_file_name;index:idx_unique_file_name_folder_id,unique"`
	LastModified int64    `gorm:"size:64;not null"`
	FolderID     int64    `gorm:"size:64;not null;index:idx_file_folder_id;index:idx_unique_file_name_folder_id,unique"`
	Folder       folderV1 `gorm:"constraint:fk_file_folder_id,OnDelete:CASCADE,OnUpdate:NO ACTION"`
}

func (*fileV1) TableName() string {
	return "metadata_files"
}

func v1Up(tx *gorm.DB) error {
	modelsToMigrate := []interface{}{
		&folderV1{},
		&fileV1{},
	}
	return tx.AutoMigrate(modelsToMigrate...)
}

func v1Down(tx *gorm.DB) error {
	modelsToMigrate := []interface{}{
		&folderV1{},
		&fileV1{},
	}
	return tx.Migrator().DropTable(modelsToMigrate...)
}

func getV1Migration() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: mignationV1ID,
		Migrate: func(tx *gorm.DB) error {
			return v1Up(tx)
		},
		Rollback: func(tx *gorm.DB) error {
			return v1Down(tx)
		},
	}
}
