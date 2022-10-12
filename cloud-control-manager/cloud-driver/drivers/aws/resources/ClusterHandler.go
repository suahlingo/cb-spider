package resources

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/aws/aws-sdk-go/service/iam"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
)

type AwsClusterHandler struct {
	Region      idrv.RegionInfo
	Client      *eks.EKS
	Iam         *iam.IAM
	AutoScaling *autoscaling.AutoScaling
}

const (
	NODEGROUP_TAG string = "nodegroup"
)

//------ Cluster Management

/*
	AWS Cluster는 Role이 필수임.
	우선, roleName=spider-eks-role로 설정, 생성 시 Role의 ARN을 조회하여 사용
*/

// ------ Cluster Management
func (ClusterHandler *AwsClusterHandler) CreateCluster(clusterReqInfo irs.ClusterInfo) (irs.ClusterInfo, error) {
	// validation check

	reqSecurityGroupIds := clusterReqInfo.Network.SecurityGroupIIDs
	var securityGroupIds []*string
	for _, securityGroupIID := range reqSecurityGroupIds {
		securityGroupIds = append(securityGroupIds, aws.String(securityGroupIID.SystemId))
	}

	reqSubnetIds := clusterReqInfo.Network.SubnetIID
	var subnetIds []*string
	for _, subnetIID := range reqSubnetIds {
		subnetIds = append(subnetIds, aws.String(subnetIID.SystemId))
	}

	//AWS의 경우 사전에 Role의 생성이 필요하며, 현재는 role 이름을 다음 이름으로 일치 시킨다.(추후 개선)
	//예시) cluster : cloud-barista-spider-eks-cluster-role, Node : cloud-barista-spider-eks-nodegroup-role
	eksRoleName := "cloud-barista-spider-eks-cluster-role"
	// get Role Arn
	eksRole, err := ClusterHandler.getRole(irs.IID{SystemId: eksRoleName})
	if err != nil {
		// role 은 required 임.
		return irs.ClusterInfo{}, err
	}
	roleArn := eksRole.Role.Arn

	reqK8sVersion := clusterReqInfo.Version

	// create cluster
	input := &eks.CreateClusterInput{
		Name: aws.String(clusterReqInfo.IId.NameId),
		ResourcesVpcConfig: &eks.VpcConfigRequest{
			SecurityGroupIds: securityGroupIds,
			SubnetIds:        subnetIds,
		},
		//RoleArn: aws.String("arn:aws:iam::012345678910:role/eks-service-role-AWSServiceRoleForAmazonEKS-J7ONKE3BQ4PI"),
		//RoleArn: aws.String(roleArn),
		RoleArn: roleArn,
		Version: aws.String(reqK8sVersion),
	}

	result, err := ClusterHandler.Client.CreateCluster(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case eks.ErrCodeResourceInUseException:
				fmt.Println(eks.ErrCodeResourceInUseException, aerr.Error())
			case eks.ErrCodeResourceLimitExceededException:
				fmt.Println(eks.ErrCodeResourceLimitExceededException, aerr.Error())
			case eks.ErrCodeInvalidParameterException:
				fmt.Println(eks.ErrCodeInvalidParameterException, aerr.Error())
			case eks.ErrCodeClientException:
				fmt.Println(eks.ErrCodeClientException, aerr.Error())
			case eks.ErrCodeServerException:
				fmt.Println(eks.ErrCodeServerException, aerr.Error())
			case eks.ErrCodeServiceUnavailableException:
				fmt.Println(eks.ErrCodeServiceUnavailableException, aerr.Error())
			case eks.ErrCodeUnsupportedAvailabilityZoneException:
				fmt.Println(eks.ErrCodeUnsupportedAvailabilityZoneException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.

			fmt.Println(err.Error())
		}
		return irs.ClusterInfo{}, err
	}

	//if cblogger.Level.String() == "debug" {
	spew.Dump(result)
	//}

	//----- wait until Status=COMPLETE -----//  :  cluster describe .status 로 확인

	clusterIID := irs.IID{NameId: clusterReqInfo.IId.NameId, SystemId: result.Cluster.Identity.String()}
	nodeGroupInfoList := clusterReqInfo.NodeGroupList
	for _, nodeGroupInfo := range nodeGroupInfoList {
		resultNodeGroupInfo, nodeGroupErr := ClusterHandler.AddNodeGroup(clusterIID, nodeGroupInfo)
		if nodeGroupErr != nil {
			fmt.Println(err.Error())
		}
		spew.Dump(resultNodeGroupInfo)
	}

	//----- wait until Status=COMPLETE -----//  :  Nodegroup이 모두 생성되면 조회

	clusterInfo, errClusterInfo := ClusterHandler.GetCluster(clusterReqInfo.IId)
	if errClusterInfo != nil {
		cblogger.Error(errClusterInfo.Error())
		return irs.ClusterInfo{}, errClusterInfo
	}
	return clusterInfo, nil
}
func (ClusterHandler *AwsClusterHandler) ListCluster() ([]*irs.ClusterInfo, error) {
	//return irs.ClusterInfo{}, nil

	input := &eks.ListClustersInput{}
	if ClusterHandler == nil {
		fmt.Println(" ClusterHandlerIs nil")
	}
	fmt.Println(ClusterHandler)
	if ClusterHandler.Client == nil {
		fmt.Println(" ClusterHandler.Client Is nil")
	}
	result, err := ClusterHandler.Client.ListClusters(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case eks.ErrCodeInvalidParameterException:
				fmt.Println(eks.ErrCodeInvalidParameterException, aerr.Error())
			case eks.ErrCodeClientException:
				fmt.Println(eks.ErrCodeClientException, aerr.Error())
			case eks.ErrCodeServerException:
				fmt.Println(eks.ErrCodeServerException, aerr.Error())
			case eks.ErrCodeServiceUnavailableException:
				fmt.Println(eks.ErrCodeServiceUnavailableException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return nil, err
	}

	spew.Dump(result)
	clusterList := []*irs.ClusterInfo{}
	for _, clusterName := range result.Clusters {

		clusterInfo, err := ClusterHandler.GetCluster(irs.IID{SystemId: *clusterName})
		if err != nil {
			continue //	에러가 나면 일단 skip시킴.
		}
		clusterList = append(clusterList, &clusterInfo)

	}
	return clusterList, nil

}
func (ClusterHandler *AwsClusterHandler) GetCluster(clusterIID irs.IID) (irs.ClusterInfo, error) {
	input := &eks.DescribeClusterInput{
		Name: aws.String(clusterIID.SystemId),
	}

	result, err := ClusterHandler.Client.DescribeCluster(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case eks.ErrCodeResourceNotFoundException:
				fmt.Println(eks.ErrCodeResourceNotFoundException, aerr.Error())
			case eks.ErrCodeClientException:
				fmt.Println(eks.ErrCodeClientException, aerr.Error())
			case eks.ErrCodeServerException:
				fmt.Println(eks.ErrCodeServerException, aerr.Error())
			case eks.ErrCodeServiceUnavailableException:
				fmt.Println(eks.ErrCodeServiceUnavailableException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return irs.ClusterInfo{}, err
	}
	spew.Dump(result)
	return irs.ClusterInfo{}, nil
}
func (ClusterHandler *AwsClusterHandler) DeleteCluster(clusterIID irs.IID) (bool, error) {
	input := &eks.DeleteClusterInput{
		Name: aws.String(clusterIID.SystemId),
	}

	result, err := ClusterHandler.Client.DeleteCluster(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case eks.ErrCodeResourceInUseException:
				fmt.Println(eks.ErrCodeResourceInUseException, aerr.Error())
			case eks.ErrCodeResourceNotFoundException:
				fmt.Println(eks.ErrCodeResourceNotFoundException, aerr.Error())
			case eks.ErrCodeClientException:
				fmt.Println(eks.ErrCodeClientException, aerr.Error())
			case eks.ErrCodeServerException:
				fmt.Println(eks.ErrCodeServerException, aerr.Error())
			case eks.ErrCodeServiceUnavailableException:
				fmt.Println(eks.ErrCodeServiceUnavailableException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return false, nil
	}
	spew.Dump(result)
	waitInput := &eks.DescribeClusterInput{
		Name: aws.String(clusterIID.SystemId),
	}
	err = ClusterHandler.Client.WaitUntilClusterDeleted(waitInput)
	if err != nil {
		return false, err
	}
	return true, nil
}

// ------ NodeGroup Management

/*
Cluster.NetworkInfo 설정과 동일 서브넷으로 설정
NodeGroup 추가시에는 대상 Cluster 정보 획득하여 설정
NodeGroup에 다른 Subnet 설정이 꼭 필요시 추후 재논의
*/
func (ClusterHandler *AwsClusterHandler) AddNodeGroup(clusterIID irs.IID, nodeGroupReqInfo irs.NodeGroupInfo) (irs.NodeGroupInfo, error) {
	// validation check
	if nodeGroupReqInfo.MaxNodeSize < 1 { // nodeGroupReqInfo.MaxNodeSize 는 최소가 1이다.
		return irs.NodeGroupInfo{}, awserr.New(CUSTOM_ERR_CODE_BAD_REQUEST, "max는 최소가 1이다.", nil)
	}

	//eksRoleName := "arn:aws:iam::050864702683:role/cb-eks-nodegroup-role"
	eksRoleName := "arn:aws:iam::050864702683:role/AWSServiceRoleForAmazonEKSNodegroup"
	//AWSServiceRoleForAmazonEKSNodegroup

	clusterInfo, err := ClusterHandler.GetCluster(clusterIID)
	if err != nil {
		return irs.NodeGroupInfo{}, err
	}
	networkInfo := clusterInfo.Network
	var subnetList []*string
	for _, subnet := range networkInfo.SubnetIID {
		subnetList = append(subnetList, &subnet.SystemId)
	}

	var nodeSecurityGroupList []*string
	for _, securityGroup := range networkInfo.SecurityGroupIIDs {
		nodeSecurityGroupList = append(nodeSecurityGroupList, &securityGroup.SystemId)
	}

	tags := map[string]string{}
	tags["key"] = NODEGROUP_TAG
	tags["value"] = nodeGroupReqInfo.IId.NameId

	input := &eks.CreateNodegroupInput{
		//AmiType: "", // Valid Values: AL2_x86_64 | AL2_x86_64_GPU | AL2_ARM_64 | CUSTOM | BOTTLEROCKET_ARM_64 | BOTTLEROCKET_x86_64, Required: No
		//CapacityType: aws.String("ON_DEMAND"),//Valid Values: ON_DEMAND | SPOT, Required: No

		ClusterName:   aws.String("cb-eks-cluster"),              //uri, required
		NodegroupName: aws.String(nodeGroupReqInfo.IId.SystemId), // required
		Tags:          aws.StringMap(tags),
		NodeRole:      aws.String(eksRoleName), // roleName, required
		ScalingConfig: &eks.NodegroupScalingConfig{
			DesiredSize: aws.Int64(int64(nodeGroupReqInfo.DesiredNodeSize)),
			MaxSize:     aws.Int64(int64(nodeGroupReqInfo.MaxNodeSize)),
			MinSize:     aws.Int64(int64(nodeGroupReqInfo.MinNodeSize)),
		},
		Subnets: subnetList,

		//DiskSize: 0,
		//InstanceTypes: ["",""],
		//Labels : {"key": "value"},
		//LaunchTemplate: {
		//	Id: "",
		//	Name: "",
		//	Version: ""
		//},

		//ReleaseVersion: "",
		RemoteAccess: &eks.RemoteAccessConfig{
			Ec2SshKey:            &nodeGroupReqInfo.KeyPairIID.SystemId,
			SourceSecurityGroups: nodeSecurityGroupList,
		},

		//Taints: [{
		//	Effect:"",
		//	Key : "",
		//	Value :""
		//}],
		//UpdateConfig: {
		//	MaxUnavailable: 0,
		//	MaxUnavailablePercentage: 0
		//},
		//Version: ""
	}

	// 필수 외에 넣을 항목들 set
	rootDiskSize, _ := strconv.ParseInt(nodeGroupReqInfo.RootDiskSize, 10, 64)
	if rootDiskSize > 0 {
		input.DiskSize = aws.Int64(rootDiskSize)
	}

	if !strings.EqualFold(nodeGroupReqInfo.VMSpecName, "") {
		var nodeSpec []string
		nodeSpec = append(nodeSpec, nodeGroupReqInfo.VMSpecName) //"p2.xlarge"
		input.InstanceTypes = aws.StringSlice(nodeSpec)
	}

	result, err := ClusterHandler.Client.CreateNodegroup(input) // 비동기
	if err != nil {
		return irs.NodeGroupInfo{}, err
	}

	spew.Dump(result)
	nodegroupName := result.Nodegroup.NodegroupName

	nodeGroup, err := ClusterHandler.GetNodeGroup(clusterIID, irs.IID{NameId: nodeGroupReqInfo.IId.NameId, SystemId: *nodegroupName})
	if err != nil {
		return irs.NodeGroupInfo{}, err
	}
	return nodeGroup, nil
}
func (ClusterHandler *AwsClusterHandler) ListNodeGroup(clusterIID irs.IID) ([]*irs.NodeGroupInfo, error) {
	input := &eks.ListNodegroupsInput{
		ClusterName: aws.String(clusterIID.SystemId),
	}
	spew.Dump(input)

	result, err := ClusterHandler.Client.ListNodegroups(input)
	if err != nil {
		return nil, err
	}
	spew.Dump(result)
	nodeGroupInfoList := []*irs.NodeGroupInfo{}
	for _, nodeGroupName := range result.Nodegroups {
		nodeGroupInfo, err := ClusterHandler.GetNodeGroup(clusterIID, irs.IID{SystemId: *nodeGroupName})
		if err != nil {
			//return nil, err
			continue
		}
		nodeGroupInfoList = append(nodeGroupInfoList, &nodeGroupInfo)
	}
	return nodeGroupInfoList, nil
}

func (ClusterHandler *AwsClusterHandler) GetNodeGroup(clusterIID irs.IID, nodeGroupIID irs.IID) (irs.NodeGroupInfo, error) {
	input := &eks.DescribeNodegroupInput{
		//AmiType: "", // Valid Values: AL2_x86_64 | AL2_x86_64_GPU | AL2_ARM_64 | CUSTOM | BOTTLEROCKET_ARM_64 | BOTTLEROCKET_x86_64, Required: No
		//CapacityType: aws.String("ON_DEMAND"),//Valid Values: ON_DEMAND | SPOT, Required: No

		ClusterName:   aws.String(clusterIID.SystemId),   //required
		NodegroupName: aws.String(nodeGroupIID.SystemId), // required
	}
	spew.Dump(input)

	result, err := ClusterHandler.Client.DescribeNodegroup(input)
	if err != nil {
		return irs.NodeGroupInfo{}, err
	}

	nodeGroupInfo, err := ClusterHandler.convertNodeGroup(result)
	if err != nil {
		return irs.NodeGroupInfo{}, err
	}
	return nodeGroupInfo, nil
}

/*
AutoScaling 이라는 별도의 메뉴가 있음.
*/
func (ClusterHandler *AwsClusterHandler) SetNodeGroupAutoScaling(clusterIID irs.IID, nodeGroupIID irs.IID, on bool) (bool, error) {

	return false, nil
}
func (ClusterHandler *AwsClusterHandler) ChangeNodeGroupScaling(clusterIID irs.IID, nodeGroupIID irs.IID,
	DesiredNodeSize int, MinNodeSize int, MaxNodeSize int) (irs.NodeGroupInfo, error) {

	// clusterIID로 cluster 정보를 조회
	// nodeGroupIID로 nodeGroup 정보를 조회
	// 		nodeGroup에 AutoScaling 그룹 이름이 있음.

	// TODO : 공통으로 뺄 것
	input := &eks.DescribeNodegroupInput{
		ClusterName:   aws.String(clusterIID.SystemId),   //required
		NodegroupName: aws.String(nodeGroupIID.SystemId), // required
	}

	result, err := ClusterHandler.Client.DescribeNodegroup(input)
	if err != nil {
		return irs.NodeGroupInfo{}, err
	}

	nodeGroupName := result.Nodegroup.NodegroupName
	nodeGroupResources := result.Nodegroup.Resources.AutoScalingGroups
	for _, autoScalingGroup := range nodeGroupResources {
		input := &autoscaling.UpdateAutoScalingGroupInput{
			AutoScalingGroupName: aws.String(*autoScalingGroup.Name),

			MaxSize:         aws.Int64(int64(MaxNodeSize)),
			MinSize:         aws.Int64(int64(MinNodeSize)),
			DesiredCapacity: aws.Int64(int64(DesiredNodeSize)),
		}

		updateResult, err := ClusterHandler.AutoScaling.UpdateAutoScalingGroup(input)
		if err != nil {
			return irs.NodeGroupInfo{}, err
		}
		spew.Dump(updateResult)

	}

	nodeGroupInfo, err := ClusterHandler.GetNodeGroup(clusterIID, irs.IID{SystemId: *nodeGroupName})
	if err != nil {
		return irs.NodeGroupInfo{}, err
	}
	return nodeGroupInfo, nil
}

func (ClusterHandler *AwsClusterHandler) RemoveNodeGroup(clusterIID irs.IID, nodeGroupIID irs.IID) (bool, error) {

	input := &eks.DeleteNodegroupInput{
		ClusterName:   aws.String(clusterIID.SystemId),   //required
		NodegroupName: aws.String(nodeGroupIID.SystemId), // required
	}

	result, err := ClusterHandler.Client.DeleteNodegroup(input)
	if err != nil {
		return false, err
	}

	spew.Dump(result.Nodegroup)

	return true, nil
}

// ------ Upgrade K8S
func (ClusterHandler *AwsClusterHandler) UpgradeCluster(clusterIID irs.IID, newVersion string) (irs.ClusterInfo, error) {
	// -- version 만 update인 경우
	input := &eks.UpdateClusterVersionInput{
		Name:    aws.String(clusterIID.SystemId),
		Version: aws.String(newVersion),
	}
	result, err := ClusterHandler.Client.UpdateClusterVersion(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case eks.ErrCodeInvalidParameterException:
				fmt.Println(eks.ErrCodeInvalidParameterException, aerr.Error())
			case eks.ErrCodeClientException:
				fmt.Println(eks.ErrCodeClientException, aerr.Error())
			case eks.ErrCodeResourceNotFoundException:
				fmt.Println(eks.ErrCodeResourceNotFoundException, aerr.Error())
			case eks.ErrCodeServerException:
				fmt.Println(eks.ErrCodeServerException, aerr.Error())
			case eks.ErrCodeInvalidRequestException:
				fmt.Println(eks.ErrCodeInvalidRequestException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
	}
	spew.Dump(result)
	// getClusterInfo
	return irs.ClusterInfo{}, nil

}

func (ClusterHandler *AwsClusterHandler) getRole(role irs.IID) (*iam.GetRoleOutput, error) {
	input := &iam.GetRoleInput{
		RoleName: aws.String(role.SystemId),
	}

	result, err := ClusterHandler.Iam.GetRole(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case iam.ErrCodeNoSuchEntityException:
				fmt.Println(iam.ErrCodeNoSuchEntityException, aerr.Error())
			case iam.ErrCodeServiceFailureException:
				fmt.Println(iam.ErrCodeServiceFailureException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return nil, err
	}

	return result, nil
}

/*
EKS의 NodeGroup정보를 Spider의 NodeGroup으로 변경
*/
func (NodeGroupHandler *AwsClusterHandler) convertNodeGroup(nodeGroupOutput *eks.DescribeNodegroupOutput) (irs.NodeGroupInfo, error) {

	nodeGroupInfo := irs.NodeGroupInfo{}

	PrintToJson(nodeGroupOutput)

	nodeGroup := nodeGroupOutput.Nodegroup
	//nodeRole := nodeGroup.NodeRole
	//version := nodeGroup.Version
	//releaseVersion := nodeGroup.ReleaseVersion

	//subnetList := nodeGroup.Subnets
	//nodeGroupStatus := nodeGroup.Status
	instanceTypeList := nodeGroup.InstanceTypes // spec

	//nodes := nodeGroup.Health.Issues[0].ResourceIds // 문제 있는 node들만 있는것이 아닌지..
	rootDiskSize := nodeGroup.DiskSize
	//nodeGroup.Taints// 미사용
	nodeGroupTagList := nodeGroup.Tags
	scalingConfig := nodeGroup.ScalingConfig
	//nodeGroup.RemoteAccess
	nodeGroupName := nodeGroup.NodegroupName

	//nodeGroup.LaunchTemplate //미사용
	//clusterName := nodeGroup.ClusterName
	//capacityType := nodeGroup.CapacityType // "ON_DEMAND"
	//amiType := nodeGroup.AmiType	// AL2_x86_64"
	//createTime := nodeGroup.CreatedAt
	//health := nodeGroup.Health // Code, Message, ResourceIds	// ,"Health":{"Issues":[{"Code":"NodeCreationFailure","Message":"Unhealthy nodes in the kubernetes cluster",
	//labelList := nodeGroup.Labels
	//nodeGroupArn := nodeGroup.NodegroupArn
	//nodeGroupResources := nodeGroup.Resources
	//nodeGroupResources.AutoScalingGroups// 미사용
	//nodeGroupResources.RemoteAccessSecurityGroup// 미사용

	nodes := []irs.IID{}
	for _, issue := range nodeGroup.Health.Issues {
		resourceIds := issue.ResourceIds
		for _, resourceId := range resourceIds {
			nodes = append(nodes, irs.IID{SystemId: *resourceId})
		}
	}

	nodeGroupInfo.NodeList = nodes
	nodeGroupInfo.MaxNodeSize = int(*scalingConfig.MaxSize)
	nodeGroupInfo.MinNodeSize = int(*scalingConfig.MinSize)

	if nodeGroupTagList == nil {
		nodeGroupTagList[NODEGROUP_TAG] = nodeGroupName // 값이없으면 nodeGroupName이랑 같은값으로 set.
	}
	nodeGroupTag := ""
	for key, val := range nodeGroupTagList {
		if strings.EqualFold("key", NODEGROUP_TAG) {
			nodeGroupTag = *val
			break
		}
		cblogger.Info(key, *val)
	}
	//printToJson(nodeGroupTagList)
	cblogger.Info("nodeGroupName=", *nodeGroupName)
	cblogger.Info("tag=", nodeGroupTagList[NODEGROUP_TAG])
	nodeGroupInfo.IId = irs.IID{
		NameId:   nodeGroupTag, // TAG에 이름
		SystemId: *nodeGroupName,
	}
	nodeGroupInfo.VMSpecName = *instanceTypeList[0]
	//nodeGroupInfo.ImageIID
	//nodeGroupInfo.KeyPairIID // keypair setting 해야하네?
	//nodeGroupInfo.RootDiskSize = strconv.FormatInt(*nodeGroup.DiskSize, 10)
	nodeGroupInfo.RootDiskSize = strconv.FormatInt(*rootDiskSize, 10)

	// TODO : node 목록 NodegroupArn 으로 조회해야하나??
	nodeList := []irs.IID{}
	//if nodeList != nil {
	//	for _, nodeId := range nodes {
	//		nodeList = append(nodeList, irs.IID{NameId: "", SystemId: *nodeId})
	//	}
	//}
	nodeGroupInfo.NodeList = nodeList
	cblogger.Info("NodeGroup")
	//	{"Nodegroup":
	//		{"AmiType":"AL2_x86_64"
	//		,"CapacityType":"ON_DEMAND"
	//		,"ClusterName":"cb-eks-cluster"
	//		,"CreatedAt":"2022-08-05T01:51:49.673Z"
	//		,"DiskSize":20
	//		,"Health":{
	//					"Issues":[
	//							{"Code":"NodeCreationFailure"
	//							,"Message":"Unhealthy nodes in the kubernetes cluster"
	//							,"ResourceIds":["i-06ee95583f3f7de5c","i-0a283a92dcce27aa8"]}]},
	//		"InstanceTypes":["t3.medium"],
	//		"Labels":{},
	//		"LaunchTemplate":null,
	//		"ModifiedAt":"2022-08-05T02:15:14.308Z",
	//		"NodeRole":"arn:aws:iam::050864702683:role/cb-eks-nodegroup-role",
	//		"NodegroupArn":"arn:aws:eks:ap-northeast-2:050864702683:nodegroup/cb-eks-cluster/cb-eks-nodegroup-test/fec135d9-c812-8862-e3b0-7b773ce70d2e","NodegroupName":"cb-eks-nodegro
	//up-test",
	//		"ReleaseVersion":"1.22.9-20220725",
	//		"RemoteAccess":{"Ec2SshKey":"cb-webtool","SourceSecurityGroups":["sg-04607666"]},
	//		"Resources":{"AutoScalingGroups":[{"Name":"eks-cb-eks-nodegroup-test-fec135d9-c812-8862-e3b0-7b773ce70d2e"}],
	//		"RemoteAccessSecurityGroup":null},
	//		"ScalingConfig":{"DesiredSize":2,"MaxSize":2,"MinSize":2},
	//		"Status":"CREATE_FAILED",
	//		"Subnets":["subnet-262d6d7a","subnet-d0ee6fab","subnet-875a62cb","subnet-e08f5b8b"],
	//		"Tags":{},
	//		"Taints":null,
	//		"UpdateConfig":{"MaxUnavailable":1,"MaxUnavailablePercentage":null},
	//		"Version":"1.22"}}

	//nodeGroupArn
	// arn format
	//arn:partition:service:region:account-id:resource-id
	//arn:partition:service:region:account-id:resource-type/resource-id
	//arn:partition:service:region:account-id:resource-type:resource-id

	PrintToJson(nodeGroupInfo)
	//return irs.NodeGroupInfo{}, awserr.New(CUSTOM_ERR_CODE_BAD_REQUEST, "추출 오류", nil)
	return nodeGroupInfo, nil
}
