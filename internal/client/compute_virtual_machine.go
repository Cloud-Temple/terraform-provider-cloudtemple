package client

import "context"

type VirtualMachineClient struct {
	c *Client
}

func (c *Compute) VirtualMachine() *VirtualMachineClient {
	return &VirtualMachineClient{c.c}
}

type VirtualMachine struct {
	ID                             string
	Name                           string
	Moref                          string
	MachineManagerType             string
	MachineManagerId               string
	MachineManagerName             string
	DatastoreName                  string
	ConsolidationNeeded            bool
	Template                       bool
	PowerState                     string
	HardwareVersion                string
	NumCoresPerSocket              int
	OperatingSystemName            string
	Cpu                            int
	CpuHotAddEnabled               bool
	CpuHotRemoveEnabled            bool
	MemoryHotAddEnabled            bool
	Memory                         int
	CpuUsage                       int
	MemoryUsage                    int
	Tools                          string
	ToolsVersion                   int
	VirtualDatacenterId            string
	DistributedVirtualPortGroupIds []string
	SppMode                        string
	Snapshoted                     bool
	TriggeredAlarms                []string
	ReplicationConfig              VirtualMachineReplicationConfig
	ExtraConfig                    []VirtualMachineExtraConfig
	Storage                        VirtualMachineStorage
	BootOptions                    VirtualMachineBootOptions
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
	Firmware         string
	BootDelay        int
	EnterBIOSSetup   bool
	BootRetryEnabled bool
	BootRetryDelay   int
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
	r := v.c.newRequest("GET", "/api/compute/v1/vcenters/virtual_machines")
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

func (v *VirtualMachineClient) Read(ctx context.Context, id string) (*VirtualMachine, error) {
	r := v.c.newRequest("GET", "/api/compute/v1/vcenters/virtual_machines/"+id)
	resp, err := v.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out VirtualMachine
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}
