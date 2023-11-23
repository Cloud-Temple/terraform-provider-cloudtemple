package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func buildGuestOSCustomizationRequest(ctx context.Context, d *schema.ResourceData) *client.CustomizeGuestOSRequest {
	dnsServerList := []string{}
	for _, policy := range d.Get("customize.0.network_config.0.dns_server_list").(*schema.Set).List() {
		dnsServerList = append(dnsServerList, policy.(string))
	}

	dnsSuffixList := []string{}
	for _, policy := range d.Get("customize.0.network_config.0.dns_suffix_list").(*schema.Set).List() {
		dnsSuffixList = append(dnsSuffixList, policy.(string))
	}

	adaptersConfig := []*client.CustomAdapterConfig{}
	for _, adapter := range d.Get("customize.0.network_config.0.adapters").([]interface{}) {
		adaptersConfig = append(adaptersConfig, &client.CustomAdapterConfig{
			MacAddress: adapter.(map[string]interface{})["mac_address"].(string),
			IpAddress:  adapter.(map[string]interface{})["ip_address"].(string),
			SubnetMask: adapter.(map[string]interface{})["subnet_mask"].(string),
			Gateway:    adapter.(map[string]interface{})["gateway"].(string),
		})
	}

	customizationRequest := &client.CustomizeGuestOSRequest{
		NetworkConfig: &client.CustomGuestNetworkConfig{
			Hostname:      d.Get("customize.0.network_config.0.hostname").(string),
			Domain:        d.Get("customize.0.network_config.0.domain").(string),
			DnsServerList: dnsServerList,
			DnsSuffixList: dnsSuffixList,
			Adapters:      adaptersConfig,
		},
	}

	if len(d.Get("customize.0.windows_config").([]interface{})) > 0 {
		customizationRequest.WindowsConfig = &client.CustomGuestWindowsConfig{
			AutoLogon:           d.Get("customize.0.windows_config.0.auto_logon").(bool),
			AutoLogonCount:      d.Get("customize.0.windows_config.0.auto_logon_count").(int),
			TimeZone:            d.Get("customize.0.windows_config.0.timezone").(int),
			Password:            d.Get("customize.0.windows_config.0.password").(string),
			JoinDomain:          d.Get("customize.0.windows_config.0.domain.0.name").(string),
			DomainAdmin:         d.Get("customize.0.windows_config.0.domain.0.admin_username").(string),
			DomainAdminPassword: d.Get("customize.0.windows_config.0.domain.0.admin_password").(string),
		}
	}

	return customizationRequest
}

func getPowerRecommendation(vm *client.VirtualMachine, powerState string, ctx context.Context, c *client.Client) (*client.VirtualMachinePowerRecommendation, error) {
	var err error
	var recommendations []*client.VirtualMachinePowerRecommendation

	if powerState == "on" || powerState == "running" {
		recommendations, err = c.Compute().VirtualMachine().Recommendation(ctx, &client.VirtualMachineRecommendationFilter{
			Id:            vm.ID,
			DatacenterId:  vm.DatacenterId,
			HostClusterId: vm.HostClusterId,
		})
		if err != nil {
			return nil, err
		}
	}

	var recommendation *client.VirtualMachinePowerRecommendation
	if len(recommendations) > 0 {
		recommendation = &client.VirtualMachinePowerRecommendation{
			Key:           recommendations[0].Key,
			HostClusterId: recommendations[0].HostClusterId,
		}
	} else {
		recommendation = nil
	}

	return recommendation, err
}

func updateNestedMapItems(d *schema.ResourceData, nestedMapItems []interface{}, key string) []interface{} {
	nestedMaps := make([]interface{}, len(nestedMapItems))

	for i, mapItems := range nestedMapItems {
		nestedMaps[i] = updateMapItems(d, mapItems, key, i)
	}

	return nestedMaps
}

func updateMapItems(d *schema.ResourceData, mapItems interface{}, key string, index int) interface{} {
	m := mapItems.(map[string]interface{})

	if value, ok := d.GetOk(fmt.Sprintf("%s.%d", key, index)); ok {

		for k, v := range value.(map[string]interface{}) {
			if _, ok := d.GetOk(fmt.Sprintf("%s.%d.%s", key, index, k)); ok {
				m[k] = v
			}
		}
	}
	return m
}

func flattenOSDisksData(osDisks []*client.VirtualDisk) []interface{} {
	if osDisks != nil {
		disks := make([]interface{}, len(osDisks))

		for i, osDisk := range osDisks {
			disks[i] = flattenOSDiskData(osDisk)
		}

		return disks
	}

	return make([]interface{}, 0)
}

func flattenOSDiskData(osDisk *client.VirtualDisk) interface{} {
	disk := make(map[string]interface{})

	disk["id"] = osDisk.ID
	disk["machine_manager_id"] = osDisk.MachineManagerId
	disk["name"] = osDisk.Name
	disk["capacity"] = osDisk.Capacity
	disk["disk_unit_number"] = osDisk.DiskUnitNumber
	disk["controller_bus_number"] = osDisk.ControllerBusNumber
	disk["datastore_id"] = osDisk.DatastoreId
	disk["datastore_name"] = osDisk.DatastoreName
	disk["instant_access"] = osDisk.InstantAccess
	disk["native_id"] = osDisk.NativeId
	disk["disk_path"] = osDisk.DiskPath
	disk["provisioning_type"] = osDisk.ProvisioningType
	disk["disk_mode"] = osDisk.DiskMode
	disk["editable"] = osDisk.Editable

	return disk
}

func flattenOSNetworkAdaptersData(osNetworkAdapters []*client.NetworkAdapter) []interface{} {
	if osNetworkAdapters != nil {
		networkAdapters := make([]interface{}, len(osNetworkAdapters))

		for i, osNetworkAdapter := range osNetworkAdapters {
			networkAdapters[i] = flattenOSNetworkAdapterData(osNetworkAdapter)
		}

		return networkAdapters
	}

	return make([]interface{}, 0)
}

func flattenOSNetworkAdapterData(osNetworkAdapter *client.NetworkAdapter) interface{} {
	networkAdapter := make(map[string]interface{})

	networkAdapter["id"] = osNetworkAdapter.ID
	networkAdapter["name"] = osNetworkAdapter.Name
	networkAdapter["network_id"] = osNetworkAdapter.NetworkId
	networkAdapter["type"] = osNetworkAdapter.Type
	networkAdapter["mac_type"] = osNetworkAdapter.MacType
	networkAdapter["mac_address"] = osNetworkAdapter.MacAddress
	networkAdapter["connected"] = osNetworkAdapter.Connected
	networkAdapter["auto_connect"] = osNetworkAdapter.AutoConnect

	return networkAdapter
}
