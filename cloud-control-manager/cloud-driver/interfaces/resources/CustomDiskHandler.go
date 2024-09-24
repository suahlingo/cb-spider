// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Resouces interfaces of Cloud Driver.
//
// by CB-Spider Team, 2022.08.

package resources

import "time"

// -------- Const
type CustomDiskStatus string

const (
	CustomDiskCreating  DiskStatus = "Creating"
	CustomDiskAvailable DiskStatus = "Available"
	CustomDiskAttached  DiskStatus = "Attached"
	CustomDiskDeleting  DiskStatus = "Deleting"
	CustomDiskError     DiskStatus = "Error"
)

// -------- Info Structure
// DiskInfo represents the information of a Disk resource.
type CustomDiskInfo struct {
	IId  IID    `json:"IId" validate:"required"`                       // {NameId, SystemId}
	Zone string `json:"Zone" validate:"required" example:"us-east-1a"` // Target Zone Name

	DiskType string `json:"DiskType" validate:"required" example:"gp2"` // "gp2", "Premium SSD", ...
	DiskSize string `json:"DiskSize" validate:"required" example:"100"` // "default", "50", "1000" (unit is GB)

	Status  DiskStatus `json:"Status" validate:"required" example:"Available"`
	OwnerVM IID        `json:"OwnerVM" validate:"omitempty"` // When the Status is DiskAttached

	CreatedTime  time.Time  `json:"CreatedTime" validate:"required"`             // The time when the disk was created
	TagList      []KeyValue `json:"TagList,omitempty" validate:"omitempty"`      // A list of tags associated with this disk
	KeyValueList []KeyValue `json:"KeyValueList,omitempty" validate:"omitempty"` // Additional key-value pairs associated with this disk
}

// -------- Disk API
type CustomDiskHandler interface {

	//------ Disk Management
	CustomCreateDisk(DiskReqInfo CustomDiskInfo) (CustomDiskInfo, error)
	CustomListDisk() ([]*CustomDiskInfo, error)
	CustomGetDisk(diskIID IID) (CustomDiskInfo, error)
	CustomChangeDiskSize(diskIID IID, size string) (bool, error)
	CustomDeleteDisk(diskIID IID) (bool, error)

	//------ Disk Attachment
	CustomAttachDisk(diskIID IID, ownerVM IID) (CustomDiskInfo, error)
	CustomDetachDisk(diskIID IID, ownerVM IID) (bool, error)
}
