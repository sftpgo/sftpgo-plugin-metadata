package db

type File struct {
	ID           int64 `gorm:"primarykey"`
	Name         string
	LastModified int64
	FolderID     int64
	Folder       Folder // foreign key
}

func (*File) TableName() string {
	return "metadata_files"
}
