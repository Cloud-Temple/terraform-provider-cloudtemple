package helpers

import (
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

func TestIsPlatformManagedDisk(t *testing.T) {
	const vmID = "vm-1"

	tests := []struct {
		name string
		disk *client.OpenIaaSVirtualDisk
		want bool
	}{
		{
			name: "read-only XO config drive on this VM is excluded",
			disk: &client.OpenIaaSVirtualDisk{
				Name: cloudConfigDriveName,
				VirtualMachines: []client.OpenIaaSVirtualDiskConnection{
					{ID: vmID, ReadOnly: true},
				},
			},
			want: true,
		},
		{
			name: "writable user disk with the colliding name stays managed",
			disk: &client.OpenIaaSVirtualDisk{
				Name: cloudConfigDriveName,
				VirtualMachines: []client.OpenIaaSVirtualDiskConnection{
					{ID: vmID, ReadOnly: false},
				},
			},
			want: false,
		},
		{
			name: "read-only disk with another name stays managed",
			disk: &client.OpenIaaSVirtualDisk{
				Name: "data-disk",
				VirtualMachines: []client.OpenIaaSVirtualDiskConnection{
					{ID: vmID, ReadOnly: true},
				},
			},
			want: false,
		},
		{
			name: "XO-named disk read-only on another VM only stays managed",
			disk: &client.OpenIaaSVirtualDisk{
				Name: cloudConfigDriveName,
				VirtualMachines: []client.OpenIaaSVirtualDiskConnection{
					{ID: "vm-other", ReadOnly: true},
				},
			},
			want: false,
		},
		{
			name: "XO-named disk without any VBD stays managed (fail-safe)",
			disk: &client.OpenIaaSVirtualDisk{
				Name: cloudConfigDriveName,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isPlatformManagedDisk(tt.disk, vmID); got != tt.want {
				t.Errorf("isPlatformManagedDisk() = %v, want %v", got, tt.want)
			}
		})
	}
}
