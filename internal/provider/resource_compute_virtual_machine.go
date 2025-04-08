package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/sethvargo/go-retry"
)

func resourceVirtualMachine() *schema.Resource {
	return &schema.Resource{
		Description: `Provision a virtual machine. This allows instances to be created, updated, and deleted.

 Virtual machines can be created using three different methods:

  - by creating a new instance with ` + "`guest_operating_system_moref`" + `
  - by cloning an existing virtual machine with ` + "`clone_virtual_machine_id`" + `
  - by deploying a content library item with ` + "`content_library_id` and `content_library_item_id`",

		CreateWithoutTimeout: computeVirtualMachineCreate,
		ReadContext:          computeVirtualMachineRead,
		UpdateContext:        computeVirtualMachineUpdate,
		DeleteContext:        computeVirtualMachineDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			// In
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"content_library_id": {
				Type:          schema.TypeString,
				Description:   "The ID of the content library to clone from. Conflict with `clone_virtual_machine_id`.",
				Optional:      true,
				ForceNew:      true,
				RequiredWith:  []string{"content_library_item_id"},
				ConflictsWith: []string{"clone_virtual_machine_id"},
				ValidateFunc:  validation.IsUUID,
			},
			"content_library_item_id": {
				Type:          schema.TypeString,
				Description:   "The ID of the content library item to clone. Conflict with `clone_virtual_machine_id`.",
				Optional:      true,
				ForceNew:      true,
				RequiredWith:  []string{"content_library_id"},
				ConflictsWith: []string{"clone_virtual_machine_id"},
				AtLeastOneOf:  []string{"clone_virtual_machine_id", "guest_operating_system_moref", "content_library_item_id"},
				ValidateFunc:  validation.IsUUID,
			},
			"deploy_options": {
				Type:          schema.TypeMap,
				ForceNew:      true,
				Optional:      true,
				ConflictsWith: []string{"clone_virtual_machine_id"},

				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"cloud_init": {
				Type:     schema.TypeMap,
				Optional: true,
				Description: `A set of cloud-init compatible key/value used to configure the virtual machine.
					
	List of cloud-init compatible keys :
	- ` + "`user-data` (This value should be base64 encoded)" + `
	- ` + "`network-config` (This value should be base64 encoded)" + `
	- ` + "`public-keys` Indicates that the instance should populate the default user's 'authorized_keys' with this value" + `
	- ` + "`instance-id`" + `
	- ` + "`password` If set, the default user's password will be set to this value to allow password based login.  The password will be good for only a single login.  If set to the string 'RANDOM' then a random password will be generated, and written to the console." + `
	- ` + "`hostname` Specifies the hostname for the appliance" + `
	- ` + "`seedfrom`" + `
	
	If you need more informations, please refer to the cloud-init documentation about the OVF datasource.

	NB : The cloud-init configuration is only triggered at virtual machine first startup and requires a cloud-init compatible OVF.
	For exemple, you can use this [Ubuntu Cloud Image](https://cloud-images.ubuntu.com/) and convert it to an OVF.
				`,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				ValidateDiagFunc: validation.MapKeyMatch(regexp.MustCompile(strings.Join([]string{
					"^user-data$",
					"^network-config$",
					"^public-keys$",
					"^instance-id$",
					"^password$",
					"^hostname$",
					"^seedfrom$"},
					"|")), `The following key is not allowed for cloud-init`),
			},
			"customize": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "Customizes a virtual machine's guest operating system. (VMWare Tools has to be installed)",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"network_config": {
							Type:        schema.TypeList,
							Required:    true,
							Description: "A collection of global network settings.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"hostname": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "The network host name of the virtual machine.",
									},
									"domain": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "The fully qualified domain name.",
									},
									"dns_server_list": {
										Type:        schema.TypeSet,
										Optional:    true,
										Description: "List of DNS servers",
										Elem: &schema.Schema{
											Type:         schema.TypeString,
											ValidateFunc: validation.IsIPAddress,
										},
									},
									"dns_suffix_list": {
										Type:        schema.TypeSet,
										Optional:    true,
										Description: "List of name resolution suffixes for the virtual network adapter. This list applies to both Windows and Linux guest customization.",
										Elem: &schema.Schema{
											Type: schema.TypeString,
										},
									},
									"adapters": {
										Type:        schema.TypeList,
										Optional:    true,
										Description: "The IP settings for the associated virtual network adapter.",
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"mac_address": {
													Type:         schema.TypeString,
													Optional:     true,
													ValidateFunc: validation.IsMACAddress,
													Description:  "The MAC address of a network adapter being customized.",
												},
												"ip_address": {
													Type:         schema.TypeString,
													Required:     true,
													ValidateFunc: validation.IsIPAddress,
													Description:  "Static IP Address for the virtual network adapter.",
												},
												"subnet_mask": {
													Type:         schema.TypeString,
													Required:     true,
													ValidateFunc: validation.IsIPAddress,
													Description:  "Subnet mask for this virtual network adapter.",
												},
												"gateway": {
													Type:         schema.TypeString,
													Required:     true,
													ValidateFunc: validation.IsIPAddress,
													Description:  "Gateway address for this virtual network adapter.",
												},
											},
										},
									},
								},
							},
						},
						"windows_config": {
							Type:        schema.TypeList,
							MaxItems:    1,
							Optional:    true,
							Description: "A set of Windows specific configurations.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"auto_logon": {
										Type:     schema.TypeBool,
										Required: true,
										Description: `
										Flag to determine whether or not the machine automatically logs on as Administrator. See also the password property.
										If the AutoLogon flag is set, password must not be blank or the guest customization will fail.`,
									},
									"auto_logon_count": {
										Type:        schema.TypeInt,
										Required:    true,
										Description: "If the AutoLogon flag is set, then the AutoLogonCount property specifies the number of times the machine should automatically log on as Administrator. Generally it should be 1, but if your setup requires a number of reboots, you may want to increase it.",
									},
									"timezone": {
										Type:        schema.TypeInt,
										Required:    true,
										Description: "The time zone index for the virtual machine. Numbers correspond to time zones listed at [ Microsoft Time Zone Index Values](https://learn.microsoft.com/en-us/previous-versions/windows/embedded/ms912391(v=winembedded.11)).",
									},
									"password": {
										Type:     schema.TypeString,
										Optional: true,
										Description: `The new administrator password for the machine. To specify that the password should be set to blank (that is, no password), set the password value to NULL. Because of encryption, "" is NOT a valid value.
										If password is set to blank and autoLogon is set, the guest customization will fail.`,
										Sensitive: true,
									},
									"domain": {
										Type:          schema.TypeList,
										MaxItems:      1,
										Optional:      true,
										Description:   "The domain identification informations to provide to the Windows guest os.",
										ConflictsWith: []string{"customize.0.windows_config.0.workgroup"},
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"name": {
													Type:        schema.TypeString,
													Optional:    true,
													Description: "The domain that the virtual machine should join. If this value is supplied, then admin_username and admin_password must also be supplied, and the workgroup name must be empty.",
													RequiredWith: []string{
														"customize.0.windows_config.0.domain.0.admin_username",
														"customize.0.windows_config.0.domain.0.admin_password",
													},
												},
												"admin_username": {
													Type:        schema.TypeString,
													Optional:    true,
													Description: "This is the domain user account used for authentication if the virtual machine is joining a domain. The user does not need to be a domain administrator, but the account must have the privileges required to add computers to the domain.",
													RequiredWith: []string{
														"customize.0.windows_config.0.domain.0.name",
														"customize.0.windows_config.0.domain.0.admin_password",
													},
												},
												"admin_password": {
													Type:        schema.TypeString,
													Optional:    true,
													Description: "This is the password for the domain user account used for authentication if the virtual machine is joining a domain.",
													Sensitive:   true,
													RequiredWith: []string{
														"customize.0.windows_config.0.domain.0.admin_username",
														"customize.0.windows_config.0.domain.0.name",
													},
												},
											},
										},
									},
									"workgroup": {
										Type:          schema.TypeString,
										Optional:      true,
										Description:   "The workgroup that the virtual machine should join. If this value is supplied, then the domain name and authentication fields must be empty.",
										ConflictsWith: []string{"customize.0.windows_config.0.domain"},
										AtLeastOneOf:  []string{"customize.0.windows_config.0.domain", "customize.0.windows_config.0.workgroup"},
									},
								},
							},
						},
					},
				},
			},
			"clone_virtual_machine_id": {
				Type:          schema.TypeString,
				Description:   "The ID of the virtual machine to clone. Conflict with `content_library_item_id`.",
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"content_library_item_id"},
				AtLeastOneOf:  []string{"clone_virtual_machine_id", "guest_operating_system_moref", "content_library_item_id"},
				ValidateFunc:  validation.IsUUID,
			},
			"guest_operating_system_moref": {
				Type:         schema.TypeString,
				Description:  "The operating system to launch the virtual machine with.",
				Optional:     true,
				Computed:     true,
				AtLeastOneOf: []string{"clone_virtual_machine_id", "guest_operating_system_moref", "content_library_item_id"},
			},
			"datacenter_id": {
				Type:         schema.TypeString,
				Description:  "The datacenter to start the virtual machine in.",
				Required:     true,
				ValidateFunc: validation.IsUUID,
			},
			"host_id": {
				Type:         schema.TypeString,
				Description:  "The host to start the virtual machine on.",
				Optional:     true,
				ValidateFunc: validation.IsUUID,
			},
			"host_cluster_id": {
				Type:         schema.TypeString,
				Description:  "The host cluster to start the virtual machine on.",
				Required:     true,
				ValidateFunc: validation.IsUUID,
			},
			"datastore_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsUUID,
			},
			"datastore_cluster_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsUUID,
			},
			"memory": {
				Type:        schema.TypeInt,
				Description: "The quantity of memory to start the virtual machine with.",
				Optional:    true,
				Default:     33554432,
			},
			"cpu": {
				Type:        schema.TypeInt,
				Description: "The number of CPUs to start the virtual machine with.",
				Optional:    true,
				Default:     1,
			},
			"num_cores_per_socket": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  1,
			},
			"cpu_hot_add_enabled": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"cpu_hot_remove_enabled": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"memory_hot_add_enabled": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"expose_hardware_virtualization": {
				Type:        schema.TypeBool,
				Description: "Enable nested hardware virtualization on the virtual machine, facilitating nested virtualization in the guest operating system (Default: false)",
				Optional:    true,
				Default:     false,
			},
			"power_state": {
				Type:         schema.TypeString,
				Description:  "Whether to start the virtual machine.",
				Optional:     true,
				Default:      "off",
				ValidateFunc: validation.StringInSlice([]string{"on", "off"}, false),
			},
			"tags": {
				Type:        schema.TypeMap,
				Description: "The tags to attach to the virtual machine.",
				Optional:    true,

				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"backup_sla_policies": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "The IDs of the SLA policies to assign to the virtual machine.",
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.IsUUID,
				},
			},
			"disks_provisioning_type": {
				Type:         schema.TypeString,
				Description:  "Overrides the provisioning type for the os_disks of an OVF.",
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice([]string{"dynamic", "staticImmediate", "staticDiffered"}, false),
			},
			"os_disk": {
				Type:        schema.TypeList,
				Description: "OS disks created from content lib item deployment or virtual machine clone.",
				Optional:    true,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						// In
						"capacity": {
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
						},
						"disk_mode": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},

						// Out
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"machine_manager_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"disk_unit_number": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"controller_bus_number": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"datastore_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"datastore_name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"instant_access": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"native_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"disk_path": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"provisioning_type": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"editable": {
							Type:     schema.TypeBool,
							Computed: true,
						},
					},
				},
			},
			"os_network_adapter": {
				Type:        schema.TypeList,
				Description: "OS network adapters created from content lib item deployment or virtual machine clone.",
				Optional:    true,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						// In
						"network_id": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
						"mac_address": {
							Type:         schema.TypeString,
							Optional:     true,
							Computed:     true,
							ComputedWhen: []string{"mac_type"},
						},
						"mac_type": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
						"auto_connect": {
							Type:     schema.TypeBool,
							Optional: true,
							Computed: true,
						},
						"connected": {
							Type:     schema.TypeBool,
							Optional: true,
							Computed: true,
						},

						// Out
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"type": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"boot_options": {
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				Computed: true,

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"firmware": {
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
							Description: "Firmware type. (BIOS or EFI)",
						},
						"boot_delay": {
							Type:        schema.TypeInt,
							Optional:    true,
							Computed:    true,
							Description: "Delay in milliseconds before starting the boot sequence. The boot delay specifies a time interval between virtual machine power on or restart and the beginning of the boot sequence.",
						},
						"enter_bios_setup": {
							Type:        schema.TypeBool,
							Optional:    true,
							Computed:    true,
							Description: "If set to true, the virtual machine automatically enters BIOS setup the next time it boots. The virtual machine resets this flag to false so that subsequent boots proceed normally.",
						},
						"boot_retry_enabled": {
							Type:        schema.TypeBool,
							Optional:    true,
							Computed:    true,
							Description: "If set to true, a virtual machine that fails to boot will try again after the bootRetryDelay time period has expired. When false, the virtual machine waits indefinitely for you to initiate boot retry.",
						},
						"boot_retry_delay": {
							Type:        schema.TypeInt,
							Optional:    true,
							Computed:    true,
							Description: "Delay in milliseconds before a boot retry. The boot retry delay specifies a time interval between virtual machine boot failure and the subsequent attempt to boot again. The virtual machine uses this value only if bootRetryEnabled is true.",
						},
						"efi_secure_boot_enabled": {
							Type:        schema.TypeBool,
							Optional:    true,
							Computed:    true,
							Description: "If set to true, the virtual machine's firmware will perform signature checks of any EFI images loaded during startup, and will refuse to start any images which do not pass those signature checks.",
						},
					},
				},
			},

			// Out
			"moref": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"machine_manager_type": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"machine_manager_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"machine_manager_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"datastore_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"datastore_cluster_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"host_cluster_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"datacenter_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"consolidation_needed": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"template": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"hardware_version": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"operating_system_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"cpu_usage": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"memory_usage": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"tools": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"tools_version": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"distributed_virtual_port_group_ids": {
				Type:     schema.TypeList,
				Computed: true,

				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"spp_mode": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"snapshoted": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"triggered_alarms": {
				Type:     schema.TypeList,
				Computed: true,

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"status": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"replication_config": {
				Type:     schema.TypeList,
				Computed: true,

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"generation": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"vm_replication_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"rpo": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"quiesce_guest_enabled": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"paused": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"opp_updates_enabled": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"net_compression_enabled": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"net_encryption_enabled": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"encryption_destination": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"disk": {
							Type:     schema.TypeList,
							Computed: true,

							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"key": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"disk_replication_id": {
										Type:     schema.TypeString,
										Computed: true,
									},
								},
							},
						},
					},
				},
			},
			"extra_config": {
				Type:     schema.TypeList,
				Computed: true,

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"value": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"storage": {
				Type:     schema.TypeList,
				Computed: true,

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"committed": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"uncommitted": {
							Type:     schema.TypeInt,
							Computed: true,
						},
					},
				},
			},
		},
		CustomizeDiff: customdiff.All(
			customdiff.ValidateChange("os_disk", func(ctx context.Context, old, new, meta any) error {
				o := len(old.([]interface{}))
				n := len(new.([]interface{}))
				if n > o && o > 0 {
					return fmt.Errorf("new os_disk blocks are not allowed if that exceeds the number of existing OS disks (%d > %d)", n, o)
				}
				return nil
			}),
			customdiff.ValidateChange("os_network_adapter", func(ctx context.Context, old, new, meta any) error {
				o := len(old.([]interface{}))
				n := len(new.([]interface{}))
				if n > o && o > 0 {
					return fmt.Errorf("new os_network_adapter blocks are not allowed if that exceeds the number of existing OS network adapters (%d > %d)", n, o)
				}
				return nil
			}),
		),
	}
}

func computeVirtualMachineCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)
	name := d.Get("name").(string)

	var activityId string
	var err error
	cloneVirtualMachineId := d.Get("clone_virtual_machine_id").(string)
	contentLibraryItemId := d.Get("content_library_item_id").(string)

	if cloneVirtualMachineId != "" {
		activityId, err = c.Compute().VirtualMachine().Clone(ctx, &client.CloneVirtualMachineRequest{
			Name:             name,
			VirtualMachineId: cloneVirtualMachineId,
			// PowerOn:           d.Get("power_state").(string) == "on",
			DatacenterId:      d.Get("datacenter_id").(string),
			HostClusterId:     d.Get("host_cluster_id").(string),
			HostId:            d.Get("host_id").(string),
			DatatoreClusterId: d.Get("datastore_cluster_id").(string),
			DatastoreId:       d.Get("datastore_id").(string),
		})
		if err != nil {
			return diag.Errorf("failed to clone virtual machine: %s", err)
		}

		activity, err := c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
		setIdFromActivityState(d, activity)
		if err != nil {
			return diag.Errorf("failed to clone virtual machine, %s", err)
		}

	} else if contentLibraryItemId != "" {
		datastoreId := d.Get("datastore_id").(string)
		if datastoreId == "" {
			return diag.Errorf("'datastore_id' is required when 'content_library_item_id' is used.")
		}

		var deployOptions []*client.DeployOption
		for k, v := range d.Get("deploy_options").(map[string]interface{}) {
			deployOptions = append(deployOptions, &client.DeployOption{
				ID:    k,
				Value: v.(string),
			})
		}

		for k, v := range d.Get("cloud_init").(map[string]interface{}) {
			if !exists(deployOptions, func(i *client.DeployOption) bool {
				return i.ID == k
			}) {
				deployOptions = append(deployOptions, &client.DeployOption{
					ID:    k,
					Value: v.(string),
				})
			}
		}

		activityId, err = c.Compute().ContentLibrary().Deploy(ctx, &client.ComputeContentLibraryItemDeployRequest{
			Name:                  name,
			ContentLibraryId:      d.Get("content_library_id").(string),
			ContentLibraryItemId:  d.Get("content_library_item_id").(string),
			HostClusterId:         d.Get("host_cluster_id").(string),
			HostId:                d.Get("host_id").(string),
			DatastoreId:           d.Get("datastore_id").(string),
			DatacenterId:          d.Get("datacenter_id").(string),
			PowerOn:               d.Get("power_state").(string) == "on",
			DisksProvisioningType: d.Get("disks_provisioning_type").(string),
			DeployOptions:         deployOptions,
		})
		if err != nil {
			return diag.Errorf("failed to deploy content library item: %s", err)
		}

		activity, err := c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
		setIdFromActivityState(d, activity)
		if err != nil {
			return diag.Errorf("failed to deploy content library item: %s", err)
		}

	} else {
		activityId, err = c.Compute().VirtualMachine().Create(ctx, &client.CreateVirtualMachineRequest{
			Name:                      name,
			DatacenterId:              d.Get("datacenter_id").(string),
			HostId:                    d.Get("host_id").(string),
			HostClusterId:             d.Get("host_cluster_id").(string),
			DatastoreId:               d.Get("datastore_id").(string),
			DatastoreClusterId:        d.Get("datastore_cluster_id").(string),
			Memory:                    d.Get("memory").(int),
			CPU:                       d.Get("cpu").(int),
			GuestOperatingSystemMoref: d.Get("guest_operating_system_moref").(string),
		})
		if err != nil {
			return diag.Errorf("failed to create virtual machine, %s", err)
		}

		activity, err := c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
		setIdFromActivityConcernedItems(d, activity, "virtual_machine")
		if err != nil {
			return diag.Errorf("failed to create virtual machine: %s", err)
		}
	}

	disks, err := c.Compute().VirtualDisk().List(ctx, d.Id())
	if err != nil {
		return diag.Errorf("failed to retrieve OS disks: %s", err)
	}

	// Overwrite with the desired config
	osDisks := updateNestedMapItems(d, flattenOSDisksData(disks), "os_disk")

	if err := d.Set("os_disk", osDisks); err != nil {
		return diag.FromErr(err)
	}

	networkAdapters, err := c.Compute().NetworkAdapter().List(ctx, d.Id())
	if err != nil {
		return diag.Errorf("failed to retrieve OS network adapters: %s", err)
	}

	// Overwrite with the desired config
	osNetworkAdapters := updateNestedMapItems(d, flattenOSNetworkAdaptersData(networkAdapters), "os_network_adapter")

	if err := d.Set("os_network_adapter", osNetworkAdapters); err != nil {
		return diag.FromErr(err)
	}

	if len(d.Get("customize").([]interface{})) > 0 {
		customizationRequest := buildGuestOSCustomizationRequest(ctx, d)
		activityId, err = c.Compute().VirtualMachine().CustomizeGuestOS(ctx, d.Id(), customizationRequest)
		if err != nil {
			return diag.Errorf("failed to customize virtual machine guest os: %s", err)
		}

		_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
		if err != nil {
			return diag.Errorf("an error has occured while customizing virtual machine guest os, %s", err)
		}
	}

	if len(d.Get("backup_sla_policies").(*schema.Set).List()) > 0 {
		// First we need to update the catalog
		jobs, err := c.Backup().Job().List(ctx, &client.BackupJobFilter{
			Type: "catalog",
		})
		if err != nil {
			return diag.Errorf("failed to find catalog job: %s", err)
		}

		var job = &client.BackupJob{}
		for _, currJob := range jobs {
			if currJob.Name == "Hypervisor Inventory" {
				job = currJob
			}
		}

		activityId, err := c.Backup().Job().Run(ctx, &client.BackupJobRunRequest{
			JobId: job.ID,
		})
		if err != nil {
			return diag.Errorf("failed to update catalog: %s", err)
		}

		_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
		if err != nil {
			return diag.Errorf("failed to update catalog, %s", err)
		}

		_, err = c.Backup().VirtualMachine().WaitForInventory(ctx, d.Id(), getWaiterOptions(ctx))
		if err != nil {
			return diag.Errorf("failed to find virtual machine in backup inventory : %s", err)
		}

		slaPolicies := []string{}
		for _, policy := range d.Get("backup_sla_policies").(*schema.Set).List() {
			slaPolicies = append(slaPolicies, policy.(string))
		}
		activityId, err = c.Backup().SLAPolicy().AssignVirtualMachine(ctx, &client.BackupAssignVirtualMachineRequest{
			VirtualMachineIds: []string{d.Id()},
			SLAPolicies:       slaPolicies,
		})
		if err != nil {
			return diag.Errorf("failed to assign policies to virtual machine, %s", err)
		}

		_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
		if err != nil {
			return diag.Errorf("failed to assign policies to virtual machine, %s", err)
		}
	}

	return updateVirtualMachine(ctx, d, meta, d.Get("power_state").(string) == "on", true)
}

func computeVirtualMachineRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	reader := readFullResource(func(ctx context.Context, c *client.Client, d *schema.ResourceData, sw *stateWriter) (interface{}, error) {
		id := d.Id()
		vm, err := c.Compute().VirtualMachine().Read(ctx, id)
		if err != nil {
			return nil, err
		}
		if vm == nil {
			return nil, nil
		}

		// Normalize the power state so that we can use it as input
		switch vm.PowerState {
		case "running":
			vm.PowerState = "on"
		case "stopped":
			vm.PowerState = "off"
		default:
			return nil, fmt.Errorf("unknown power state %q", vm.PowerState)
		}

		// Normalize the backup_sla_policies
		slaPolicies, err := c.Backup().SLAPolicy().List(ctx, &client.BackupSLAPolicyFilter{
			VirtualMachineId: d.Id(),
		})
		if err != nil {
			return nil, err
		}

		slaPoliciesIds := []string{}
		for _, slaPolicy := range slaPolicies {
			slaPoliciesIds = append(slaPoliciesIds, slaPolicy.ID)
		}

		sw.set("backup_sla_policies", slaPoliciesIds)

		osDisks := []interface{}{}
		for _, osDisk := range d.Get("os_disk").([]interface{}) {
			if osDisk == nil {
				continue
			}
			osDiskId := osDisk.(map[string]interface{})["id"].(string)
			if osDiskId != "" {
				disk, err := c.Compute().VirtualDisk().Read(ctx, osDiskId)
				if err != nil {
					return nil, err
				}
				osDisks = append(osDisks, flattenOSDiskData(disk))
			}
		}

		sw.set("os_disk", osDisks)

		osNetworkAdapters := []interface{}{}
		for _, osNetworkAdapter := range d.Get("os_network_adapter").([]interface{}) {
			if osNetworkAdapter == nil {
				continue
			}
			osNetworkAdapterId := osNetworkAdapter.(map[string]interface{})["id"].(string)
			if osNetworkAdapterId != "" {
				networkAdapter, err := c.Compute().NetworkAdapter().Read(ctx, osNetworkAdapterId)
				if err != nil {
					return nil, err
				}
				osNetworkAdapters = append(osNetworkAdapters, flattenOSNetworkAdapterData(networkAdapter))
			}
		}

		sw.set("os_network_adapter", osNetworkAdapters)

		readTags(ctx, sw, c, d.Id())

		return vm, nil
	})

	return reader(ctx, d, meta)
}

func computeVirtualMachineUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	return updateVirtualMachine(ctx, d, meta, d.HasChange("power_state"), false)
}

func updateVirtualMachine(ctx context.Context, d *schema.ResourceData, meta any, updatePower bool, customizing bool) diag.Diagnostics {
	c := getClient(meta)

	req := &client.UpdateVirtualMachineRequest{
		Id:                           d.Id(),
		Ram:                          d.Get("memory").(int),
		Cpu:                          d.Get("cpu").(int),
		CorePerSocket:                d.Get("num_cores_per_socket").(int),
		HotCpuAdd:                    d.Get("cpu_hot_add_enabled").(bool),
		HotCpuRemove:                 d.Get("cpu_hot_remove_enabled").(bool),
		HotMemAdd:                    d.Get("memory_hot_add_enabled").(bool),
		ExposeHardwareVirtualization: d.Get("expose_hardware_virtualization").(bool),
	}

	if len(d.Get("boot_options").([]interface{})) > 0 {
		bootOptions := d.Get("boot_options").([]interface{})[0]
		req.BootOptions = &client.BootOptions{
			BootDelay:            bootOptions.(map[string]interface{})["boot_delay"].(int),
			BootRetryDelay:       bootOptions.(map[string]interface{})["boot_retry_delay"].(int),
			BootRetryEnabled:     bootOptions.(map[string]interface{})["boot_retry_enabled"].(bool),
			EnterBIOSSetup:       bootOptions.(map[string]interface{})["enter_bios_setup"].(bool),
			Firmware:             strings.ToLower(bootOptions.(map[string]interface{})["firmware"].(string)),
			EFISecureBootEnabled: bootOptions.(map[string]interface{})["efi_secure_boot_enabled"].(bool),
		}
	}

	activityId, err := c.Compute().VirtualMachine().Update(ctx, req)
	if err != nil {
		return diag.Errorf("failed to update virtual machine: %s", err)
	}
	_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
	if err != nil {
		return diag.Errorf("failed to update virtual machine, %s", err)
	}

	if diags := updateTags(ctx, c, d, d.Id(), "vcenter_virtual_machine", "vmware"); diags != nil {
		return diags
	}

	if d.HasChange("name") {
		activityId, err := c.Compute().VirtualMachine().Rename(ctx, d.Id(), d.Get("name").(string))
		if err != nil {
			return diag.Errorf("failed to rename virtual machine, %s", err)
		}
		_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
		if err != nil {
			return diag.Errorf("failed to rename virtual machine, %s", err)
		}
	}

	if d.HasChange("guest_operating_system_moref") {
		activityId, err := c.Compute().VirtualMachine().Guest(ctx, d.Id(), &client.UpdateGuestRequest{
			GuestOperatingSystemMoref: d.Get("guest_operating_system_moref").(string),
		})
		if err != nil {
			return diag.Errorf("failed to update virtual machine guest operating system, %s", err)
		}
		_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
		if err != nil {
			return diag.Errorf("failed to update virtual machine guest operating system, %s", err)
		}
	}

	if d.HasChange("datacenter_id") || d.HasChange("host_id") || d.HasChange("host_cluster_id") || d.HasChange("datastore_id") || d.HasChange("datastore_cluster_id") {
		activityId, err := c.Compute().VirtualMachine().Relocate(ctx, &client.RelocateVirtualMachineRequest{
			VirtualMachines:    []string{d.Id()},
			Priority:           "highPriority",
			DatacenterId:       d.Get("datacenter_id").(string),
			HostId:             d.Get("host_id").(string),
			HostClusterId:      d.Get("host_cluster_id").(string),
			DatastoreId:        d.Get("datastore_id").(string),
			DatastoreClusterId: d.Get("datastore_cluster_id").(string),
		})
		if err != nil {
			return diag.Errorf("failed to relocate virtual machine, %s", err)
		}
		_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
		if err != nil {
			return diag.Errorf("failed to relocate virtual machine, %s", err)
		}
	}

	if d.HasChange("customize") && !customizing {
		vm, err := c.Compute().VirtualMachine().Read(ctx, d.Id())
		if err != nil {
			return diag.Errorf("failed to read virtual machine: %s", err)
		}
		if vm.PowerState == "running" {
			activityId, err := c.Compute().VirtualMachine().Power(ctx, &client.PowerRequest{
				ID:           d.Id(),
				DatacenterId: vm.Datacenter.ID,
				PowerAction:  "off",
			})
			if err != nil {
				return diag.Errorf("failed to power off virtual machine: %s", err)
			}
			_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
			if err != nil {
				return diag.Errorf("failed to power off virtual machine, %s", err)
			}
		}

		customizationRequest := buildGuestOSCustomizationRequest(ctx, d)
		activityId, err = c.Compute().VirtualMachine().CustomizeGuestOS(ctx, d.Id(), customizationRequest)
		if err != nil {
			return diag.Errorf("failed to customize virtual machine guest os: %s", err)
		}

		_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
		if err != nil {
			return diag.Errorf("an error has occured while customizing virtual machine guest os, %s", err)
		}

		if vm.PowerState == "running" {
			recommendation, err := getPowerRecommendation(vm, vm.PowerState, ctx, c)
			if err != nil {
				return diag.Errorf("failed to get power recommendation for virtual machine: %s", err)
			}

			activityId, err := c.Compute().VirtualMachine().Power(ctx, &client.PowerRequest{
				ID:             d.Id(),
				DatacenterId:   vm.Datacenter.ID,
				PowerAction:    "on",
				Recommendation: recommendation,
			})
			if err != nil {
				return diag.Errorf("failed to power on virtual machine: %s", err)
			}
			_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
			if err != nil {
				return diag.Errorf("failed to power on virtual machine, %s", err)
			}
		}
	}

	if d.HasChange("backup_sla_policies") {
		backupVm, err := c.Backup().VirtualMachine().Read(ctx, d.Id())
		if err != nil {
			return diag.Errorf("failed to get sla policies of virtual machine %s, %s", d.Id(), err)
		}
		if backupVm == nil {
			jobs, err := c.Backup().Job().List(ctx, &client.BackupJobFilter{
				Type: "catalog",
			})
			if err != nil {
				return diag.Errorf("failed to find catalog job: %s", err)
			}

			var job = &client.BackupJob{}
			for _, currJob := range jobs {
				if currJob.Name == "Hypervisor Inventory" {
					job = currJob
				}
			}

			activityId, err := c.Backup().Job().Run(ctx, &client.BackupJobRunRequest{
				JobId: job.ID,
			})
			if err != nil {
				return diag.Errorf("failed to update catalog: %s", err)
			}

			_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
			if err != nil {
				return diag.Errorf("failed to update catalog, %s", err)
			}

			_, err = c.Backup().Job().WaitForCompletion(ctx, job.ID, getWaiterOptions(ctx))
			if err != nil {
				return diag.Errorf("failed to update catalog, %s", err)
			}
		}

		slaPolicies := []string{}
		for _, policy := range d.Get("backup_sla_policies").(*schema.Set).List() {
			slaPolicies = append(slaPolicies, policy.(string))
		}
		activityId, err = c.Backup().SLAPolicy().AssignVirtualMachine(ctx, &client.BackupAssignVirtualMachineRequest{
			VirtualMachineIds: []string{d.Id()},
			SLAPolicies:       slaPolicies,
		})
		if err != nil {
			return diag.Errorf("failed to assign policies to virtual machine, %s", err)
		}

		_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
		if err != nil {
			return diag.Errorf("failed to assign policies to virtual machine, %s", err)
		}
	}

	if d.HasChange("os_disk") {
		for i, osDisk := range d.Get("os_disk").([]interface{}) {
			if osDisk == nil {
				continue
			}
			disk := osDisk.(map[string]interface{})
			if disk["id"].(string) != "" && d.HasChange(fmt.Sprintf("os_disk.%d", i)) {
				activityId, err := c.Compute().VirtualDisk().Update(ctx, &client.UpdateVirtualDiskRequest{
					ID:          disk["id"].(string),
					NewCapacity: disk["capacity"].(int),
					DiskMode:    disk["disk_mode"].(string),
				})
				if err != nil {
					return diag.Errorf("failed to update virtual disk: %s", err)
				}
				_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
				if err != nil {
					return diag.Errorf("failed to update virtual disk, %s", err)
				}
			}
		}
	}

	if d.HasChange("os_network_adapter") {
		for i, osNetworkAdapter := range d.Get("os_network_adapter").([]interface{}) {
			if osNetworkAdapter == nil {
				continue
			}
			networkAdapter := osNetworkAdapter.(map[string]interface{})
			if networkAdapter["id"].(string) != "" && d.HasChange(fmt.Sprintf("os_network_adapter.%d", i)) {
				macType := networkAdapter["mac_type"].(string)
				macAddress := networkAdapter["mac_address"].(string)
				if macType == "ASSIGNED" {
					macAddress = ""
				}

				activityId, err := c.Compute().NetworkAdapter().Update(ctx, &client.UpdateNetworkAdapterRequest{
					ID:           networkAdapter["id"].(string),
					NewNetworkId: networkAdapter["network_id"].(string),
					AutoConnect:  networkAdapter["auto_connect"].(bool),
					MacAddress:   macAddress,
				})
				if err != nil {
					return diag.Errorf("failed to update network adapter, %s", err)
				}
				_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
				if err != nil {
					return diag.Errorf("failed to update network adapter, %s", err)
				}

				if d.HasChange(fmt.Sprintf("os_network_adapter.%d.connected", i)) {
					var msg string
					var action func(context.Context, string) (string, error)
					if networkAdapter["connected"].(bool) {
						msg = "connect"
						action = c.Compute().NetworkAdapter().Connect
					} else {
						msg = "disconnect"
						action = c.Compute().NetworkAdapter().Disconnect
					}

					// Connecting a network adapter can fail right after the VM has been powered
					// on so we retry here until we reach the timeout
					b := retry.NewFibonacci(1 * time.Second)
					b = retry.WithCappedDuration(20*time.Second, b)

					err = retry.Do(ctx, b, func(ctx context.Context) error {
						activityId, err = action(ctx, networkAdapter["id"].(string))
						if err != nil {
							return err
						}
						_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
						return err
					})
					if err != nil {
						return diag.Errorf("failed to %s network adapter, %s", msg, err)
					}
				}
			}
		}
	}

	if updatePower {
		powerState := d.Get("power_state").(string)

		vm, err := c.Compute().VirtualMachine().Read(ctx, d.Id())
		if err != nil {
			return diag.Errorf("failed to read virtual machine: %s", err)
		}

		recommendation, err := getPowerRecommendation(vm, powerState, ctx, c)
		if err != nil {
			return diag.Errorf("failed to get power recommendation for virtual machine: %s", err)
		}

		activityId, err = c.Compute().VirtualMachine().Power(ctx, &client.PowerRequest{
			ID:             d.Id(),
			DatacenterId:   vm.Datacenter.ID,
			PowerAction:    powerState,
			Recommendation: recommendation,
		})
		if err != nil {
			return diag.Errorf("failed to power %s virtual machine: %s", powerState, err)
		}
		_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
		if err != nil {
			return diag.Errorf("failed to power %s virtual machine, %s", powerState, err)
		}
	}

	return computeVirtualMachineRead(ctx, d, meta)
}

func computeVirtualMachineDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	vm, err := c.Compute().VirtualMachine().Read(ctx, d.Id())
	if err != nil {
		return diag.Errorf("failed to read virtual effect: %s", err)
	}

	if vm.PowerState == "running" {
		activityId, err := c.Compute().VirtualMachine().Power(ctx, &client.PowerRequest{
			ID:           d.Id(),
			DatacenterId: vm.Datacenter.ID,
			PowerAction:  "off",
		})
		if err != nil {
			return diag.Errorf("failed to power off virtual machine: %s", err)
		}
		_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
		if err != nil {
			return diag.Errorf("failed to power off virtual machine, %s", err)
		}
	}

	activityId, err := c.Compute().VirtualMachine().Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("failed to delete virtual machine: %s", err)
	}
	if _, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx)); err != nil {
		return diag.Errorf("failed to delete virtual machine, %s", err)
	}
	return nil
}
