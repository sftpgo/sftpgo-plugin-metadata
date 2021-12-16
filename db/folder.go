package db

type Folder struct {
	ID        int64 `gorm:"primarykey"`
	Path      string
	StorageID string
}

func (*Folder) TableName() string {
	return "metadata_folders"
}
