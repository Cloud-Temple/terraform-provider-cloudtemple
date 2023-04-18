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
	ID                             string                          `terraform:"id"`
	Name                           string                          `terraform:"name"`
	Moref                          string                          `terraform:"moref"`
	MachineManagerType             string                          `terraform:"machine_manager_type"`
	MachineManagerId               string                          `terraform:"machine_manager_id"`
	MachineManagerName             string                          `terraform:"machine_manager_name"`
	DatastoreName                  string                          `terraform:"datastore_name"`
	ConsolidationNeeded            bool                            `terraform:"consolidation_needed"`
	Template                       bool                            `terraform:"template"`
	PowerState                     string                          `terraform:"power_state"`
	HardwareVersion                string                          `terraform:"hardware_version"`
	NumCoresPerSocket              int                             `terraform:"num_cores_per_socket"`
	OperatingSystemName            string                          `terraform:"operating_system_name"`
	Cpu                            int                             `terraform:"cpu"`
	CpuHotAddEnabled               bool                            `terraform:"cpu_hot_add_enabled"`
	CpuHotRemoveEnabled            bool                            `terraform:"cpu_hot_remove_enabled"`
	MemoryHotAddEnabled            bool                            `terraform:"memory_hot_add_enabled"`
	Memory                         int                             `terraform:"memory"`
	CpuUsage                       int                             `terraform:"cpu_usage"`
	MemoryUsage                    int                             `terraform:"memory_usage"`
	Tools                          string                          `terraform:"tools"`
	ToolsVersion                   int                             `terraform:"tools_version"`
	DatacenterId                   string                          `terraform:"datacenter_id"`
	DistributedVirtualPortGroupIds []string                        `terraform:"distributed_virtual_port_group_ids"`
	SppMode                        string                          `terraform:"spp_mode"`
	Snapshoted                     bool                            `terraform:"snapshoted"`
	TriggeredAlarms                []VirtualMachineTriggeredAlarm  `terraform:"triggered_alarms"`
	ReplicationConfig              VirtualMachineReplicationConfig `terraform:"replication_config"`
	ExtraConfig                    []VirtualMachineExtraConfig     `terraform:"extra_config"`
	Storage                        VirtualMachineStorage           `terraform:"storage"`
	BootOptions                    VirtualMachineBootOptions       `terraform:"boot_options"`
}

type VirtualMachineTriggeredAlarm struct {
	ID     string `type:"id"`
	Status string `type:"status"`
}

type VirtualMachineReplicationConfig struct {
	Generation            int                  `terraform:"generation"`
	VmReplicationId       string               `terraform:"vm_replication_id"`
	Rpo                   int                  `terraform:"rpo"`
	QuiesceGuestEnabled   bool                 `terraform:"quiesce_guest_enabled"`
	Paused                bool                 `terraform:"paused"`
	OppUpdatesEnabled     bool                 `terraform:"opp_updates_enabled"`
	NetCompressionEnabled bool                 `terraform:"net_compression_enabled"`
	NetEncryptionEnabled  bool                 `terraform:"net_encryption_enabled"`
	EncryptionDestination bool                 `terraform:"encryption_destination"`
	Disk                  []VirtualMachineDisk `terraform:"disk"`
}

type VirtualMachineDisk struct {
	Key               int    `terraform:"key"`
	DiskReplicationId string `terraform:"disk_replication_id"`
}

type VirtualMachineExtraConfig struct {
	Key   string `terraform:"key"`
	Value string `terraform:"value"`
}

type VirtualMachineStorage struct {
	Committed   int `terraform:"committed"`
	Uncommitted int `terraform:"uncommitted"`
}

type VirtualMachineBootOptions struct {
	Firmware         string `terraform:"firmware"`
	BootDelay        int    `terraform:"boot_delay"`
	EnterBIOSSetup   bool   `terraform:"enter_bios_setup"`
	BootRetryEnabled bool   `terraform:"boot_retry_enabled"`
	BootRetryDelay   int    `terraform:"boot_retry_delay"`
}

type BootOptions struct {
	BootDelay        int    `json:"bootDelay"`
	BootRetryDelay   int    `json:"bootRetryDelay"`
	BootRetryEnabled bool   `json:"bootRetryEnabled"`
	EnterBIOSSetup   bool   `json:"enterBIOSSetup"`
	Firmware         string `json:"firmware"`
}

type PowerRequest struct {
	ID             string `json:"id,omitempty"`
	DatacenterId   string `json:"datacenterId,omitempty"`
	PowerAction    string `json:"powerAction,omitempty"`
	ForceEnterBIOS bool   `json:"forceEnterBIOS,omitempty"`
}

func (v *VirtualMachineClient) List(
	ctx context.Context,
	allOptions bool,
	machineManagerId string,
	replicated bool,
	template bool,
	datacenters []string,
	networks []string,
	datastores []string,
	hosts []string,
	vmwareToolsVersions []int) ([]*VirtualMachine, error) {

	// TODO: filters
	r := v.c.newRequest("GET", "/compute/v1/vcenters/virtual_machines")
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
	Id            string       `json:"id"`
	Ram           int          `json:"ram"`
	Cpu           int          `json:"cpu"`
	CorePerSocket int          `json:"corePerSocket"`
	HotCpuAdd     bool         `json:"hotCpuAdd"`
	HotCpuRemove  bool         `json:"hotCpuRemove"`
	HotMemAdd     bool         `json:"hotMemAdd"`
	BootOptions   *BootOptions `json:"bootOptions,omitempty"`
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
