package resources

type CustomIID struct {
	NameId   string `json:"NameId" validate:"required" example:"user-defined-name"`
	SystemId string `json:"SystemId" validate:"required" example:"csp-defined-id"`
}

type CustomSecurityInfo struct {
	IId CustomIID `json:"IId" validate:"required"` // {NameId, SystemId}

	VpcIID CustomIID `json:"VpcIID" validate:"required"` // {NameId, SystemId}

	SecurityRules *[]CustomSecurityRuleInfo `json:"SecurityRules" validate:"required" description:"A list of security rules applied to this security group"`

	TagList      []KeyValue `json:"TagList,omitempty" validate:"omitempty" description:"A list of tags associated with this security group"`
	KeyValueList []KeyValue `json:"KeyValueList,omitempty" validate:"omitempty" description:"Additional key-value pairs associated with this security group"`
}

type CustomSecurityRuleInfo struct {
	Direction  string `json:"Direction" validate:"required" example:"inbound"`         // inbound or outbound
	IPProtocol string `json:"IPProtocol" validate:"required" example:"TCP"`            // TCP, UDP, ICMP, ALL
	FromPort   string `json:"FromPort" validate:"required" example:"22"`               // TCP, UDP: 1~65535, ICMP, ALL: -1
	ToPort     string `json:"ToPort" validate:"required" example:"22"`                 // TCP, UDP: 1~65535, ICMP, ALL: -1
	CIDR       string `json:"CIDR,omitempty" validate:"omitempty" example:"0.0.0.0/0"` // if not specified, defaults to 0.0.0.0/0
}

type CustomHandler interface {
	GetVmSecurityGroups(vmIID CustomIID) ([]CustomSecurityInfo, error)
	GetSecurityRules(vmIID CustomIID) ([]CustomSecurityRuleInfo, error)
}
