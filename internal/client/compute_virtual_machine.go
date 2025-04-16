package client

import (
	"context"
)

type VirtualMachineClient struct {
	c *Client
}

func (c *ComputeClient) VirtualMachine() *VirtualMachineClient {
	return &VirtualMachineClient{c.c}
}

type VirtualMachine struct {
	ID                             string
	Name                           string
	Moref                          string
	MachineManager                 BaseObject
	Datacenter                     BaseObject
	HostCluster                    BaseObject
	Datastore                      BaseObject
	DatastoreCluster               BaseObject
	ConsolidationNeeded            bool
	Template                       bool
	PowerState                     string
	HardwareVersion                string
	NumCoresPerSocket              int
	OperatingSystemName            string
	OperatingSystemMoref           string
	Cpu                            int
	CpuHotAddEnabled               bool
	CpuHotRemoveEnabled            bool
	MemoryHotAddEnabled            bool
	Memory                         int
	CpuUsage                       int
	MemoryUsage                    int
	Tools                          string
	ToolsVersion                   int
	DistributedVirtualPortGroupIds []string
	SppMode                        string
	Snapshoted                     bool
	TriggeredAlarms                []VirtualMachineTriggeredAlarm
	ReplicationConfig              VirtualMachineReplicationConfig
	ExtraConfig                    []VirtualMachineExtraConfig
	Storage                        VirtualMachineStorage
	BootOptions                    VirtualMachineBootOptions
	ExposeHardwareVirtualization   bool
}

type VirtualMachineTriggeredAlarm struct {
	ID     string
	Status string
}

type VirtualMachineReplicationConfig struct {
	Generation            int
	VmReplicationId       string
	Rpo                   int
	QuiesceGuestEnabled   bool
	Paused                bool
	OppUpdatesEnabled     bool
	NetCompressionEnabled bool
	NetEncryptionEnabled  bool
	EncryptionDestination bool
	Disk                  []VirtualMachineDisk
}

type VirtualMachineDisk struct {
	Key               int
	DiskReplicationId string
}

type VirtualMachineExtraConfig struct {
	Key   string
	Value string
}

type VirtualMachineStorage struct {
	Committed   int
	Uncommitted int
}

type VirtualMachineBootOptions struct {
	Firmware             string
	BootDelay            int
	EnterBIOSSetup       bool
	BootRetryEnabled     bool
	BootRetryDelay       int
	EFISecureBootEnabled bool
}

type BootOptions struct {
	BootDelay            int    `json:"bootDelay"`
	BootRetryDelay       int    `json:"bootRetryDelay"`
	BootRetryEnabled     bool   `json:"bootRetryEnabled"`
	EnterBIOSSetup       bool   `json:"enterBIOSSetup"`
	Firmware             string `json:"firmware"`
	EFISecureBootEnabled bool   `json:"efiSecureBootEnabled"`
}

type PowerRequest struct {
	ID             string                             `json:"id,omitempty"`
	DatacenterId   string                             `json:"datacenterId,omitempty"`
	PowerAction    string                             `json:"powerAction,omitempty"`
	ForceEnterBIOS bool                               `json:"forceEnterBIOS,omitempty"`
	Recommendation *VirtualMachinePowerRecommendation `json:"recommendation,omitempty"`
}

type CustomAdapterConfig struct {
	MacAddress string `json:"macAddress,omitempty"`
	IpAddress  string `json:"ipAddress"`
	SubnetMask string `json:"subnetMask"`
	Gateway    string `json:"gateway"`
}

type CustomGuestNetworkConfig struct {
	Hostname      string                 `json:"hostname"`
	Domain        string                 `json:"domain"`
	DnsServerList []string               `json:"dnsServerList,omitempty"`
	DnsSuffixList []string               `json:"dnsSuffixList,omitempty"`
	Adapters      []*CustomAdapterConfig `json:"adapters,omitempty"`
}

type CustomGuestWindowsConfig struct {
	AutoLogon           bool   `json:"autoLogon"`
	AutoLogonCount      int    `json:"autoLogonCount"`
	TimeZone            int    `json:"timezone"`
	Password            string `json:"password"`
	JoinDomain          string `json:"joinDomain,omitempty"`
	DomainAdmin         string `json:"domainAdmin,omitempty"`
	DomainAdminPassword string `json:"domainAdminPassword,omitempty"`
	JoinWorkgroup       string `json:"joinWorkgroup,omitempty"`
}

type CustomizeGuestOSRequest struct {
	NetworkConfig *CustomGuestNetworkConfig `json:"networkConfig"`
	WindowsConfig *CustomGuestWindowsConfig `json:"windowsConfig,omitempty"`
}

type VirtualMachineFilter struct {
	Name             string   `filter:"name"`
	MachineManagerID string   `filter:"machineManagerId"`
	AllOptions       bool     `filter:"allOptions"`
	Datacenters      []string `filter:"datacenters"`
	Networks         []string `filter:"networks"`
	Datastores       []string `filter:"datastores"`
	Hosts            []string `filter:"hosts"`
	HostClusters     []string `filter:"hostClusters"`
}

func (v *VirtualMachineClient) List(ctx context.Context, filter *VirtualMachineFilter) ([]*VirtualMachine, error) {
	r := v.c.newRequest("GET", "/compute/v1/vcenters/virtual_machines")
	r.addFilter(filter)
	resp, err := v.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out []*VirtualMachine
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}

type CreateVirtualMachineRequest struct {
	DatacenterId              string `json:"datacenterId,omitempty"`
	HostId                    string `json:"hostId,omitempty"`
	HostClusterId             string `json:"hostClusterId,omitempty"`
	DatastoreId               string `json:"datastoreId,omitempty"`
	DatastoreClusterId        string `json:"datastoreClusterId,omitempty"`
	Name                      string `json:"name,omitempty"`
	Memory                    int    `json:"memory,omitempty"`
	CPU                       int    `json:"cpu,omitempty"`
	GuestOperatingSystemMoref string `json:"guestOperatingSystemMoref,omitempty"`
}

func (v *VirtualMachineClient) Create(ctx context.Context, req *CreateVirtualMachineRequest) (string, error) {
	r := v.c.newRequest("POST", "/compute/v1/vcenters/virtual_machines")
	r.obj = req
	return v.c.doRequestAndReturnActivity(ctx, r)
}

func (v *VirtualMachineClient) Read(ctx context.Context, id string) (*VirtualMachine, error) {
	r := v.c.newRequest("GET", "/compute/v1/vcenters/virtual_machines/%s", id)
	resp, err := v.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 403)
	if err != nil || !found {
		return nil, err
	}

	var out VirtualMachine
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

type UpdateVirtualMachineRequest struct {
	Id                           string       `json:"id"`
	Ram                          int          `json:"ram"`
	MemoryReservation            int          `json:"memoryReservation"`
	Cpu                          int          `json:"cpu"`
	CorePerSocket                int          `json:"corePerSocket"`
	HotCpuAdd                    bool         `json:"hotCpuAdd"`
	HotCpuRemove                 bool         `json:"hotCpuRemove"`
	HotMemAdd                    bool         `json:"hotMemAdd"`
	BootOptions                  *BootOptions `json:"bootOptions,omitempty"`
	ExposeHardwareVirtualization bool         `json:"exposeHardwareVirtualization,omitempty"`
}

func (v *VirtualMachineClient) Update(ctx context.Context, req *UpdateVirtualMachineRequest) (string, error) {
	r := v.c.newRequest("PATCH", "/compute/v1/vcenters/virtual_machines")
	r.obj = req
	return v.c.doRequestAndReturnActivity(ctx, r)
}

func (v *VirtualMachineClient) Delete(ctx context.Context, id string) (string, error) {
	r := v.c.newRequest("DELETE", "/compute/v1/vcenters/virtual_machines/%s", id)
	return v.c.doRequestAndReturnActivity(ctx, r)
}

func (v *VirtualMachineClient) Power(ctx context.Context, req *PowerRequest) (string, error) {
	r := v.c.newRequest("PATCH", "/compute/v1/vcenters/virtual_machines/power")
	r.obj = req
	return v.c.doRequestAndReturnActivity(ctx, r)
}

func (v *VirtualMachineClient) Rename(ctx context.Context, id string, name string) (string, error) {
	r := v.c.newRequest("PATCH", "/compute/v1/vcenters/virtual_machines/rename")
	r.obj = map[string]string{
		"id":   id,
		"name": name,
	}
	return v.c.doRequestAndReturnActivity(ctx, r)
}

type CloneVirtualMachineRequest struct {
	Name              string `json:"name"`
	VirtualMachineId  string `json:"-"`
	PowerOn           bool   `json:"powerOn"`
	DatacenterId      string `json:"datacenterId,omitempty"`
	HostClusterId     string `json:"hostClusterId,omitempty"`
	HostId            string `json:"hostId,omitempty"`
	DatatoreClusterId string `json:"datastoreClusterId,omitempty"`
	DatastoreId       string `json:"datastoreId,omitempty"`
}

func (v *VirtualMachineClient) Clone(ctx context.Context, req *CloneVirtualMachineRequest) (string, error) {
	r := v.c.newRequest("POST", "/compute/v1/vcenters/virtual_machines/%s/clone", req.VirtualMachineId)
	r.obj = req
	return v.c.doRequestAndReturnActivity(ctx, r)
}

type RelocateVirtualMachineRequest struct {
	VirtualMachines    []string         `json:"virtualMachines"`
	Priority           string           `json:"priority"`
	DatacenterId       string           `json:"datacenterId,omitempty"`
	HostId             string           `json:"hostId,omitempty"`
	HostClusterId      string           `json:"hostClusterId,omitempty"`
	DatastoreId        string           `json:"datastoreId,omitempty"`
	DatastoreClusterId string           `json:"datastoreClusterId,omitempty"`
	NetworkData        []*NetworkData   `json:"networkData,omitempty"`
	DiskPlacements     []*DiskPlacement `json:"diskPlacements,omitempty"`
}

type DiskPlacement struct {
	VirtualDiskId      string `json:"virtualDiskId"`
	VirtualMachineId   string `json:"virtualMachineId"`
	DatastoreId        string `json:"datastoreId"`
	DatastoreClusterId string `json:"datastoreClusterId"`
}

func (v *VirtualMachineClient) Relocate(ctx context.Context, req *RelocateVirtualMachineRequest) (string, error) {
	r := v.c.newRequest("POST", "/compute/v1/vcenters/virtual_machines/relocate")
	r.obj = req
	return v.c.doRequestAndReturnActivity(ctx, r)
}

type UpdateGuestRequest struct {
	GuestOperatingSystemMoref string `json:"guestOperatingSystemMoref"`
}

func (v *VirtualMachineClient) Guest(ctx context.Context, id string, req *UpdateGuestRequest) (string, error) {
	r := v.c.newRequest("PATCH", "/compute/v1/vcenters/virtual_machines/%s/guest", id)
	r.obj = req
	return v.c.doRequestAndReturnActivity(ctx, r)
}

func (v *VirtualMachineClient) CustomizeGuestOS(ctx context.Context, id string, req *CustomizeGuestOSRequest) (string, error) {
	r := v.c.newRequest("PATCH", "/compute/v1/vcenters/virtual_machines/%s/guest/customize", id)
	r.obj = req
	return v.c.doRequestAndReturnActivity(ctx, r)
}

type VirtualMachineRecommendationFilter struct {
	Id            string `filter:"virtualMachineId"`
	DatacenterId  string `filter:"datacenterId"`
	HostClusterId string `filter:"hostClusterId"`
}

type VirtualMachinePowerRecommendation struct {
	Key             int    `json:"key"`
	HostClusterId   string `json:"hostClusterId"`
	HostId          string `json:"hostId"`
	HostClusterName string `json:"hostClusterName"`
	HostName        string `json:"hostName"`
}

func (v *VirtualMachineClient) Recommendation(ctx context.Context, filter *VirtualMachineRecommendationFilter) ([]*VirtualMachinePowerRecommendation, error) {
	r := v.c.newRequest("GET", "/compute/v1/vcenters/virtual_machines/power/recommendations")
	r.addFilter(filter)
	resp, err := v.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 403)
	if err != nil || !found {
		return nil, err
	}

	var out []*VirtualMachinePowerRecommendation
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}
