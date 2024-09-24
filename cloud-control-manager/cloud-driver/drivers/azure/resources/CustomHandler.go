package resources

import (
	"context"
	"errors"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v6"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"strings"
)

type AzureCustomHandler struct {
	CredentialInfo idrv.CredentialInfo
	Region         idrv.RegionInfo
	Ctx            context.Context
	Client         *armcompute.VirtualMachinesClient
	NicClient      *armnetwork.InterfacesClient     //네트워크 정보
	SecurityClient *armnetwork.SecurityGroupsClient //보안그룹 조회
}

func (handler *AzureCustomHandler) GetVmSecurityGroups(vmIID irs.CustomIID) ([]irs.CustomSecurityInfo, error) {
	// VM IID 변환
	hiscallInfo := GetCallLogScheme(handler.Region, call.CUSTOMHANDLER, vmIID.NameId, "GetVmSecurityGroups()")
	start := call.Start()
	//vm핸들러에 있던거
	convertedIID, err := CustomConvertVMIID(vmIID, handler.CredentialInfo, handler.Region)
	if err != nil {
		return nil, fmt.Errorf("failed to convert VM IID: %v", err)
	}

	// VM 정보 가져오기
	vm, err := CustomGetRawVM(convertedIID, handler.Region.Region, handler.Client, handler.Ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get VM: %v", err)
	}

	// NIC로 네트워크 정보 가져오기
	if vm.Properties == nil || vm.Properties.NetworkProfile == nil {
		return nil, fmt.Errorf("no network profile found for VM")
	}

	networkInterfaces := vm.Properties.NetworkProfile.NetworkInterfaces

	var securityGroups []irs.CustomSecurityInfo

	for _, nic := range networkInterfaces {
		fmt.Printf("Network Interface ID: %s\n", *nic.ID)

		//NIC ID예서 실제 이름 추출
		nicIDParts := strings.Split(*nic.ID, "/") //슬래시로 분리하여 마지막부분만 추출
		nicName := nicIDParts[len(nicIDParts)-1]  //NIC의 실제 이름 출력

		nicInfo, err := handler.NicClient.Get(handler.Ctx, handler.Region.Region, nicName, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to get NIC information: %v", err)
		}

		// 보안 그룹 ID 가져오기
		if nicInfo.Properties != nil && nicInfo.Properties.NetworkSecurityGroup != nil {
			securityGroupID := *nicInfo.Properties.NetworkSecurityGroup.ID

			securityGroupIDParts := strings.Split(securityGroupID, "/")
			securityGroupName := securityGroupIDParts[len(securityGroupIDParts)-1]

			// 보안 그룹 정보 가져오기
			securityGroup, err := handler.SecurityClient.Get(handler.Ctx, handler.Region.Region, securityGroupName, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to get security group: %v", err)
			}

			securityInfo := irs.CustomSecurityInfo{
				IId: irs.CustomIID{
					NameId:   *securityGroup.Name,
					SystemId: *securityGroup.ID,
				},
				VpcIID: irs.CustomIID{
					NameId:   "",
					SystemId: "",
				},
			}

			var securityRuleArr []irs.CustomSecurityRuleInfo
			if securityGroup.Properties != nil && securityGroup.Properties.SecurityRules != nil {
				for _, sgRule := range securityGroup.Properties.SecurityRules {
					if *sgRule.Properties.Access == armnetwork.SecurityRuleAccessAllow {
						ruleInfo := irs.CustomSecurityRuleInfo{
							Direction:  string(*sgRule.Properties.Direction),    // 규칙 방향 (인바운드/아웃바운드)
							IPProtocol: string(*sgRule.Properties.Protocol),     // 프로토콜
							FromPort:   *sgRule.Properties.SourcePortRange,      // 시작 포트
							ToPort:     *sgRule.Properties.DestinationPortRange, // 종료 포트
							CIDR:       *sgRule.Properties.SourceAddressPrefix,  // CIDR
						}
						securityRuleArr = append(securityRuleArr, ruleInfo)
					}
				}
			}
			securityInfo.SecurityRules = &securityRuleArr

			if securityGroup.Tags != nil {
				securityInfo.TagList = setTagList(securityGroup.Tags)
			}

			keyValues := []irs.KeyValue{
				{Key: "ResourceGroup", Value: handler.Region.Region},
			}
			securityInfo.KeyValueList = keyValues

			securityGroups = append(securityGroups, securityInfo)
		} else {
			fmt.Println("No security group associated with the NIC.")
		}
	}

	LoggingInfo(hiscallInfo, start)
	return securityGroups, nil
}

// 보안규칙리스트
func (handler *AzureCustomHandler) GetSecurityRules(vmIID irs.CustomIID) ([]irs.CustomSecurityRuleInfo, error) {
	hiscallInfo := GetCallLogScheme(handler.Region, call.CUSTOMHANDLER, vmIID.NameId, "GetVmSecurityRules()")
	start := call.Start()

	securityInfos, err := handler.GetVmSecurityGroups(vmIID)
	if err != nil {
		return nil, fmt.Errorf("falied to get security groups for vm: %v", err)
	}

	var securityRules []irs.CustomSecurityRuleInfo

	for _, securityInfo := range securityInfos {
		if securityInfo.SecurityRules != nil {
			securityRules = append(securityRules, *securityInfo.SecurityRules...)
		}
	}

	LoggingInfo(hiscallInfo, start)
	return securityRules, nil

}

func CustomConvertVMIID(vmIID irs.CustomIID, credentialInfo idrv.CredentialInfo, regionInfo idrv.RegionInfo) (irs.CustomIID, error) {
	if vmIID.NameId == "" && vmIID.SystemId == "" {
		return vmIID, errors.New(fmt.Sprintf("nvalid IID"))
	}
	if vmIID.SystemId == "" {
		sysID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/virtualMachines/%s", credentialInfo.SubscriptionId, regionInfo.Region, vmIID.NameId)
		return irs.CustomIID{NameId: vmIID.NameId, SystemId: sysID}, nil
	} else {
		slist := strings.Split(vmIID.SystemId, "/")
		if len(slist) == 0 {
			return vmIID, errors.New(fmt.Sprintf("Invalid IID"))
		}
		s := slist[len(slist)-1]
		return irs.CustomIID{NameId: s, SystemId: vmIID.SystemId}, nil
	}
}

func CustomGetRawVM(vmIID irs.CustomIID, resourceGroup string, client *armcompute.VirtualMachinesClient, ctx context.Context) (armcompute.VirtualMachine, error) {
	if vmIID.NameId == "" {
		var vmList []*armcompute.VirtualMachine

		pager := client.NewListPager(resourceGroup, nil)

		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return armcompute.VirtualMachine{}, nil
			}

			for _, vm := range page.Value {
				vmList = append(vmList, vm)
			}
		}

		for _, vm := range vmList {
			if *vm.ID == vmIID.SystemId {
				return *vm, nil
			}
		}
		notExistVpcErr := errors.New(fmt.Sprintf("The VM id %s not found", vmIID.SystemId))
		return armcompute.VirtualMachine{}, notExistVpcErr
	} else {
		resp, err := client.Get(ctx, resourceGroup, vmIID.NameId, &armcompute.VirtualMachinesClientGetOptions{
			Expand: (*armcompute.InstanceViewTypes)(toStrPtr(string(armcompute.InstanceViewTypesInstanceView))),
		})
		if err != nil {
			return armcompute.VirtualMachine{}, err
		}

		return resp.VirtualMachine, nil
	}
}
