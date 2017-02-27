package instance

import (
	"encoding/json"
	"fmt"
	"strings"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"

	"bosh-google-cpi/api"
	subnet "bosh-google-cpi/google/subnetwork_service"
	"bosh-google-cpi/util"

	"google.golang.org/api/compute/v0.beta"
)

const defaultRootDiskSizeGb = 10
const userDataKey = "user_data"

func (i GoogleInstanceService) Create(vmProps *Properties, networks Networks, registryEndpoint string) (string, error) {
	uuidStr, err := i.uuidGen.Generate()
	if err != nil {
		return "", bosherr.WrapErrorf(err, "Generating random Google Instance name")
	}

	instanceName := vmProps.Name
	if instanceName == "" {
		instanceName = fmt.Sprintf("%s-%s", googleInstanceNamePrefix, uuidStr)
	}
	canIPForward := networks.CanIPForward()
	diskParams := i.createDiskParams(vmProps.Stemcell, vmProps.RootDiskSizeGb, vmProps.RootDiskType)
	metadataParams, err := i.createMatadataParams(instanceName, registryEndpoint, networks)
	if err != nil {
		return "", err
	}
	networkInterfacesParams, err := i.createNetworkInterfacesParams(networks, vmProps.Zone)
	if err != nil {
		return "", err
	}
	schedulingParams := i.createSchedulingParams(vmProps.AutomaticRestart, vmProps.OnHostMaintenance, vmProps.Preemptible)
	serviceAccountsParams := i.createServiceAccountsParams(vmProps)

	// Handle tags
	allTags := append(networks.Tags(), vmProps.Tags...)
	tags := compute.Tags{}
	tags.Items = allTags.Unique()

	vm := &compute.Instance{
		Name:              instanceName,
		Description:       googleInstanceDescription,
		CanIpForward:      canIPForward,
		Disks:             diskParams,
		MachineType:       vmProps.MachineType,
		Metadata:          metadataParams,
		NetworkInterfaces: networkInterfacesParams,
		Scheduling:        schedulingParams,
		ServiceAccounts:   serviceAccountsParams,
		Tags:              &tags,
	}

	if vmProps.Accelerator.Type != "" {
		acceleratorConfig := []*compute.AcceleratorConfig{
			&compute.AcceleratorConfig{
				AcceleratorType:  vmProps.Accelerator.Type,
				AcceleratorCount: vmProps.Accelerator.Count,
			},
		}
		vm.GuestAccelerators = acceleratorConfig
	}

	i.logger.Debug(googleInstanceServiceLogTag, "Creating Google Instance with params: %v", vm)
	operation, err := i.computeServiceB.Instances.Insert(i.project, util.ResourceSplitter(vmProps.Zone), vm).Do()
	if err != nil {
		i.logger.Debug(googleInstanceServiceLogTag, "Failed to create Google Instance: %v", err)
		return "", api.NewVMCreationFailedError(err.Error(), true)
	}

	if operation, err = i.operationService.WaiterB(operation, vmProps.Zone, ""); err != nil {
		i.logger.Debug(googleInstanceServiceLogTag, "Failed to create Google Instance: %v", err)
		i.CleanUp(vm.Name)
		return "", api.NewVMCreationFailedError(err.Error(), true)
	}

	if vmProps.TargetPool != "" {
		if err := i.addToTargetPool(operation.TargetLink, vmProps.TargetPool); err != nil {
			i.logger.Debug(googleInstanceServiceLogTag, "Failed to add created Google Instance to Target Pool: %v", err)
			i.CleanUp(vm.Name)
			return "", api.NewVMCreationFailedError(err.Error(), true)
		}
	}

	if vmProps.BackendService != "" {
		if err := i.addToBackendService(operation.TargetLink, vmProps.BackendService); err != nil {
			i.logger.Debug(googleInstanceServiceLogTag, "Failed to add created Google Instance to Backend Service: %v", err)
			i.CleanUp(vm.Name)
			return "", api.NewVMCreationFailedError(err.Error(), true)
		}
	}

	return vm.Name, nil
}

func (i GoogleInstanceService) CleanUp(id string) {
	if err := i.Delete(id); err != nil {
		i.logger.Debug(googleInstanceServiceLogTag, "Failed cleaning up Google Instance '%s': %v", id, err)
	}

}

func (i GoogleInstanceService) createDiskParams(stemcell string, diskSize int, diskType string) []*compute.AttachedDisk {
	var disks []*compute.AttachedDisk

	if diskSize == 0 {
		diskSize = defaultRootDiskSizeGb
	}
	disk := &compute.AttachedDisk{
		AutoDelete: true,
		Boot:       true,
		InitializeParams: &compute.AttachedDiskInitializeParams{
			DiskSizeGb:  int64(diskSize),
			DiskType:    diskType,
			SourceImage: stemcell,
		},
		Mode: "READ_WRITE",
		Type: "PERSISTENT",
	}
	disks = append(disks, disk)

	return disks
}

func (i GoogleInstanceService) createMatadataParams(name string, regEndpoint string, networks Networks) (*compute.Metadata, error) {
	serverName := GoogleUserDataServerName{Name: name}
	registryEndpoint := GoogleUserDataRegistryEndpoint{Endpoint: regEndpoint}
	userData := GoogleUserData{Server: serverName, Registry: registryEndpoint}

	if networkDNS := networks.DNS(); len(networkDNS) > 0 {
		userData.DNS = GoogleUserDataDNSItems{NameServer: networkDNS}
	}

	ud, err := json.Marshal(userData)
	if err != nil {
		return nil, bosherr.WrapErrorf(err, "Marshalling user data")
	}

	var metadataItems []*compute.MetadataItems
	userDataValue := string(ud)
	metadataItem := &compute.MetadataItems{Key: userDataKey, Value: userDataValue}
	metadataItems = append(metadataItems, metadataItem)
	metadata := &compute.Metadata{Items: metadataItems}

	return metadata, nil
}

func (i GoogleInstanceService) createNetworkInterfacesParams(networks Networks, zone string) ([]*compute.NetworkInterface, error) {
	network, found, err := i.networkService.Find(networks.NetworkName())
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, bosherr.WrapErrorf(err, "Network '%s' does not exist", networks.NetworkName())
	}

	subnetworkLink := ""
	if networks.SubnetworkName() != "" {
		subnetwork, err := i.subnetworkService.Find(networks.SubnetworkName(), util.RegionFromZone(zone))
		if err != nil {
			if err == subnet.ErrSubnetNotFound {
				return nil, bosherr.WrapErrorf(err, "Subnetwork '%s' does not exist", networks.SubnetworkName())
			}
			return nil, err
		}
		subnetworkLink = subnetwork.SelfLink
	}

	var networkInterfaces []*compute.NetworkInterface
	var accessConfigs []*compute.AccessConfig

	vipNetwork := networks.VipNetwork()
	if networks.EphemeralExternalIP() || vipNetwork.IP != "" {
		accessConfig := &compute.AccessConfig{
			Name: "External NAT",
			Type: "ONE_TO_ONE_NAT",
		}
		if vipNetwork.IP != "" {
			accessConfig.NatIP = vipNetwork.IP
		}
		accessConfigs = append(accessConfigs, accessConfig)
	}

	networkInterface := &compute.NetworkInterface{
		Network:       network.SelfLink,
		Subnetwork:    subnetworkLink,
		AccessConfigs: accessConfigs,
		NetworkIP:     networks.StaticPrivateIP(),
	}
	networkInterfaces = append(networkInterfaces, networkInterface)

	return networkInterfaces, nil
}

func (i GoogleInstanceService) createSchedulingParams(
	automaticRestart bool,
	onHostMaintenance string,
	preemptible bool,
) *compute.Scheduling {
	if preemptible {
		return &compute.Scheduling{Preemptible: preemptible}
	}

	scheduling := &compute.Scheduling{
		AutomaticRestart:  automaticRestart,
		OnHostMaintenance: onHostMaintenance,
		Preemptible:       preemptible,
	}

	if onHostMaintenance == "" {
		scheduling.OnHostMaintenance = "MIGRATE"
	}

	return scheduling
}

func (i GoogleInstanceService) createServiceAccountsParams(vmProps *Properties) []*compute.ServiceAccount {
	// No service account and no scopes, so return an empty slice.
	if vmProps.ServiceAccount == "" && len(vmProps.ServiceScopes) == 0 {
		return nil
	}

	// No service account, but scopes are specified. Set the "default" account.
	if vmProps.ServiceAccount == "" && len(vmProps.ServiceScopes) > 0 {
		vmProps.ServiceAccount = "default"
	}

	// A service account, but no scopes. Set the "full access" scope.
	if vmProps.ServiceAccount != "" && len(vmProps.ServiceScopes) == 0 {
		vmProps.ServiceScopes = ServiceScopes([]string{"https://www.googleapis.com/auth/cloud-platform"})
	}

	// Format scopes and create a slice of *compute.ServiceAccount
	var scopes []string
	for _, scope := range vmProps.ServiceScopes {
		if strings.HasPrefix(scope, "https://www.googleapis.com/auth/") {
			scopes = append(scopes, scope)
		} else {
			scopes = append(scopes, fmt.Sprintf("https://www.googleapis.com/auth/%s", scope))
		}
	}

	serviceAccount := &compute.ServiceAccount{
		Email:  string(vmProps.ServiceAccount),
		Scopes: scopes,
	}
	return []*compute.ServiceAccount{serviceAccount}
}

func (i GoogleInstanceService) addToTargetPool(instanceSelfLink string, targetPoolName string) error {
	if err := i.targetPoolService.AddInstance(targetPoolName, instanceSelfLink); err != nil {
		return err
	}

	return nil
}

func (i GoogleInstanceService) removeFromTargetPool(instanceSelfLink string) error {
	targetPool, found, err := i.targetPoolService.FindByInstance(instanceSelfLink, "")
	if err != nil {
		return err
	}

	if found {
		if err := i.targetPoolService.RemoveInstance(targetPool, instanceSelfLink); err != nil {
			return err
		}
	}

	return nil
}

func (i GoogleInstanceService) updateTargetPool(instance *compute.Instance, targetPoolName string) error {
	// Check if instance is associated to a target pool
	currentTargetPool, _, err := i.targetPoolService.FindByInstance(instance.SelfLink, "")
	if err != nil {
		return err
	}

	// Check if target pool info has changed
	if targetPoolName != currentTargetPool {
		if currentTargetPool != "" {
			if err := i.targetPoolService.RemoveInstance(currentTargetPool, instance.SelfLink); err != nil {
				return err
			}
		}

		if targetPoolName != "" {
			if err := i.targetPoolService.AddInstance(targetPoolName, instance.SelfLink); err != nil {
				return err
			}
		}
	}

	return nil
}

func (i GoogleInstanceService) addToBackendService(instanceSelfLink, backendServiceName string) error {
	if err := i.backendServiceService.AddInstance(backendServiceName, instanceSelfLink); err != nil {
		return err
	}

	return nil
}

func (i GoogleInstanceService) removeFromBackendService(instanceSelfLink string) error {
	if err := i.backendServiceService.RemoveInstance(instanceSelfLink); err != nil {
		return err
	}

	return nil
}
