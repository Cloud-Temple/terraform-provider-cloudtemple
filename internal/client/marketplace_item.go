package client

import "context"

type MarketplaceItemClient struct {
	c *Client
}

func (c *MarketplaceClient) Item() *MarketplaceItemClient {
	return &MarketplaceItemClient{c.c}
}

type MarketplaceItem struct {
	ID            string
	Name          string
	Editor        string
	Icon          string
	Description   string
	DescriptionEN string
	CreationDate  string
	LastUpdate    string
	Categories    []string
	Type          string
	Version       string
	Build         string
	Details       struct {
		Overview           string
		HowToUse           string
		Support            string
		TermsAndConditions string
	}
	DetailsEN struct {
		Overview           string
		HowToUse           string
		Support            string
		TermsAndConditions string
	}
	DeploymentOptions struct {
		Targets []struct {
			Key   string
			Name  string
			SKUs  []string
			Files []string
		}
	}
}

type MarketplaceOpenIaasItemInfo struct {
	Name        string
	Description string
	CPU         struct {
		Count          int
		CoresPerSocket int
	}
	Memory          int
	OperatingSystem struct {
		Name    string
		Distro  string
		Version string
	}
	Disks []struct {
		Name                string
		VirtualSize         int
		PhysicalUtilisation int
		Type                string
	}
	NetworkAdapters []struct {
		Name        string
		NetworkName string
		MTU         int
	}
	HighAvailability struct {
		Enabled         bool
		RestartPriority string
	}
	PVDrivers struct {
		Detected bool
		Version  string
		UpToDate bool
	}
}

type MarketplaceVMWareItemInfo struct {
	Name string
	CPU  struct {
		Count             int
		NumCoresPerSocket int
	}
	Memory          int
	OperatingSystem struct {
		ID   string
		Name string
	}
	HardwareVersion string
	NetworkAdapters []struct {
		Name        string
		NetworkName string
		Type        string
	}
	Controllers []struct {
		ID      string
		Name    string
		Type    string
		SubType string
	}
	Disks []struct {
		Name          string
		Capacity      int
		ControllerId  string
		PopulatedSize int
	}
	ExtraConfig []struct {
		Key      string
		Value    string
		Required bool
	}
}

func (v *MarketplaceItemClient) Read(ctx context.Context, id string) (*MarketplaceItem, error) {
	r := v.c.newRequest("GET", "/marketplace/v1/items/%s", id)
	resp, err := v.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 403)
	if err != nil || !found {
		return nil, err
	}

	var out MarketplaceItem
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

func (v *MarketplaceItemClient) ReadInfo(ctx context.Context, id string, target string) (*MarketplaceOpenIaasItemInfo, *MarketplaceVMWareItemInfo, error) {
	r := v.c.newRequest("GET", "/marketplace/v1/items/%s/%s/info", id, target)
	resp, err := v.c.doRequest(ctx, r)
	if err != nil {
		return nil, nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 403)
	if err != nil || !found {
		return nil, nil, err
	}

	if target == "open_iaas" {
		var out MarketplaceOpenIaasItemInfo
		if err := decodeBody(resp, &out); err != nil {
			return nil, nil, err
		}
		return &out, nil, nil
	} else if target == "vmware" {
		var out MarketplaceVMWareItemInfo
		if err := decodeBody(resp, &out); err != nil {
			return nil, nil, err
		}
		return nil, &out, nil
	}

	return nil, nil, nil
}

func (v *MarketplaceItemClient) List(ctx context.Context) ([]*MarketplaceItem, error) {
	r := v.c.newRequest("GET", "/marketplace/v1/items")
	resp, err := v.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out []*MarketplaceItem
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}

type NetworkDataMapping struct {
	SourceNetworkName    string `json:"sourceNetworkName"`
	DestinationNetworkId string `json:"destinationNetworkId"`
}

type MarketplaceOpenIaasDeployementRequest struct {
	ID                  string               `json:"id"`
	Name                string               `json:"name"`
	StorageRepositoryID string               `json:"storageRepositoryId"`
	NetworkData         []NetworkDataMapping `json:"networkData,omitempty"`
	CloudInit           CloudInit            `json:"cloudInit,omitempty"`
}

func (c *MarketplaceItemClient) DeployOpenIaasItem(ctx context.Context, req *MarketplaceOpenIaasDeployementRequest) (string, error) {
	r := c.c.newRequest("POST", "/marketplace/v1/items/open_iaas/deploy")
	r.obj = req
	return c.c.doRequestAndReturnActivity(ctx, r)
}

type MarketplaceVMWareDeployementRequest struct {
	ID            string               `json:"id"`
	Name          string               `json:"name"`
	DatacenterID  string               `json:"datacenterId"`
	HostClusterID string               `json:"hostClusterId,omitempty"`
	HostID        string               `json:"hostId,omitempty"`
	DatastoreID   string               `json:"datastoreId"`
	NetworkData   []NetworkDataMapping `json:"networkData,omitempty"`
	DeployOptions []*DeployOption      `json:"deployOptions,omitempty"`
}

func (c *MarketplaceItemClient) DeployVMWareItem(ctx context.Context, req *MarketplaceVMWareDeployementRequest) (string, error) {
	r := c.c.newRequest("POST", "/marketplace/v1/items/vmware/deploy")
	r.obj = req
	return c.c.doRequestAndReturnActivity(ctx, r)
}
