// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Resouces interfaces of Cloud Driver.
//
// by CB-Spider Team, 2023.09.

package resources

// -------- Const
type CustomZoneStatus string

const (
	CustomZoneAvailable   ZoneStatus = "Available"
	CustomZoneUnavailable ZoneStatus = "Unavailable"
	CustomNotSupported    ZoneStatus = "StatusNotSupported"
)

type CustomRegionZoneInfo struct {
	Name        string
	DisplayName string
	ZoneList    []ZoneInfo

	KeyValueList []KeyValue
}

type CustomZoneInfo struct {
	Name        string
	DisplayName string
	Status      ZoneStatus // Available | Unavailable | NotSupported

	KeyValueList []KeyValue
}

type CustomRegionZoneHandler interface {
	CustomListRegionZone() ([]*CustomRegionZoneInfo, error)

	CustomGetRegionZone(Name string) (CustomRegionZoneInfo, error)
	CustomListOrgRegion() (string, error) // return string: json format
	CustomListOrgZone() (string, error)   // return string: json format
}
