package client

import (
	"context"
	"fmt"
	"strconv"
)

type PublicCloudVMInstanceClient struct {
	c *Client
}

// Instance returns the VM instance sub-client — the core writable entity of the
// Public Cloud VM Instances product (service /vm_instances, mount
// /v1/virtual_machines). Every write is ASYNCHRONOUS: it returns 201 +
// Location:<activityId>, so the write methods return the raw activityId and the
// caller (the resource) tracks it through the shared Shiva Activities waiter,
// extracts the VM id from the completed activity and never guesses it.
func (v *PublicCloudVMClient) Instance() *PublicCloudVMInstanceClient {
	return &PublicCloudVMInstanceClient{v.c}
}

// publicCloudVMInstanceListPageSize is the API's maximum page size for
// GET /vm_instances/v1/virtual_machines (limit max = 200). List/ListStrict page
// through with limit+offset so a tenant with more than one page of VMs is never
// silently truncated (E0-6 / #414); the provider has no other pagination.
const publicCloudVMInstanceListPageSize = 200

// PublicCloudVMInstanceRef is the {id, name} shape used for the resolved
// availability zone, template, instance family and backup policy references
// returned by the API.
type PublicCloudVMInstanceRef struct {
	ID   string
	Name string
}

// PublicCloudVMInstance mirrors an element of GET /vm_instances/v1/virtual_machines
// and the GET /{id} detail. The broker camelizes responses (ramGb, disksSizeGb,
// guestToolsInstalled, ...); Go's encoding/json matches struct fields
// case-insensitively, so no json tags are needed on the read path. BackupPolicy
// is nullable in the API and decodes to nil when null.
type PublicCloudVMInstance struct {
	ID                  string
	Name                string
	Status              string
	AZ                  PublicCloudVMInstanceRef
	Template            PublicCloudVMInstanceRef
	InstanceFamily      PublicCloudVMInstanceRef
	VCPU                int
	RAMGb               int
	DisksSizeGb         int
	BackupPolicy        *PublicCloudVMInstanceRef
	GuestToolsInstalled bool
	CreatedAt           string
	UpdatedAt           string
}

// PublicCloudVMInstanceFilter carries the server-side query filters of the list
// endpoint. Only string fields are represented here because addFilter maps
// string/bool/[]string tags; the int limit/offset pagination cursor is added
// manually by the list loop. Note the query parameter for the instance family is
// `familyId` (≠ the create body's `instanceFamilyId`).
type PublicCloudVMInstanceFilter struct {
	Name               string `filter:"name"`
	Status             string `filter:"status"`
	AvailabilityZoneID string `filter:"availabilityZoneId"`
	FamilyID           string `filter:"familyId"`
	OrderBy            string `filter:"orderBy"`
	OrderDir           string `filter:"orderDir"`
}

// List returns every VM instance matching the filter, auto-paginating over the
// full result set. Pages are accepted with requireOK (200/201/206).
func (i *PublicCloudVMInstanceClient) List(ctx context.Context, filter *PublicCloudVMInstanceFilter) ([]*PublicCloudVMInstance, error) {
	return i.list(ctx, filter, false)
}

// ListStrict is List with a 200-ONLY contract on every page (206/partial is
// rejected as an error). It is the authoritative source of truth for an absence
// decision (E0-9): a VM is only accepted as deleted when a complete, strictly
// successful listing does not contain its id. A partial or access-denied listing
// must fail closed (never drop state on an incomplete listing).
func (i *PublicCloudVMInstanceClient) ListStrict(ctx context.Context, filter *PublicCloudVMInstanceFilter) ([]*PublicCloudVMInstance, error) {
	return i.list(ctx, filter, true)
}

func (i *PublicCloudVMInstanceClient) list(ctx context.Context, filter *PublicCloudVMInstanceFilter, strict bool) ([]*PublicCloudVMInstance, error) {
	var out []*PublicCloudVMInstance
	seen := make(map[string]struct{})
	for offset := 0; ; offset += publicCloudVMInstanceListPageSize {
		req := i.c.newRequest("GET", "/vm_instances/v1/virtual_machines")
		if filter != nil {
			req.addFilter(filter)
		}
		req.params.Add("limit", strconv.Itoa(publicCloudVMInstanceListPageSize))
		req.params.Add("offset", strconv.Itoa(offset))

		page, err := i.readPage(ctx, req, strict)
		if err != nil {
			return nil, err
		}

		// Deduplicate across pages and count the ids this page contributes: a
		// well-behaved server never repeats an id across offsets.
		newInPage := 0
		for _, vm := range page {
			if vm == nil {
				continue
			}
			if _, dup := seen[vm.ID]; dup {
				continue
			}
			seen[vm.ID] = struct{}{}
			out = append(out, vm)
			newInPage++
		}

		// A short (or empty) page means the collection is exhausted.
		if len(page) < publicCloudVMInstanceListPageSize {
			break
		}
		// A FULL page that contributes no new id means the server is not honouring
		// `offset` (it keeps returning the same page). Fail closed rather than loop
		// forever (unbounded memory / spin) or silently truncate — a partial
		// listing must never be mistaken for the complete set (E0-6 anti-truncation;
		// E0-9 relies on a complete listing as absence evidence).
		if newInPage == 0 {
			return nil, fmt.Errorf("virtual machine listing made no progress at offset %d (the API may be ignoring pagination); refusing to loop or truncate", offset)
		}
	}
	return out, nil
}

// readPage issues one paginated request and decodes the bare-array page,
// applying the strict (200-only) or lenient (requireOK) success contract. It
// isolates the per-request body close so the pagination loop cannot leak a
// response body.
func (i *PublicCloudVMInstanceClient) readPage(ctx context.Context, req *request, strict bool) ([]*PublicCloudVMInstance, error) {
	resp, err := i.c.doRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)

	if strict {
		if err := requireHttpCodes(resp, 200); err != nil {
			return nil, err
		}
	} else {
		if err := requireOK(resp); err != nil {
			return nil, err
		}
	}

	var page []*PublicCloudVMInstance
	if err := decodeBody(resp, &page); err != nil {
		return nil, err
	}
	return page, nil
}

// Read returns a single VM instance by UUID. A positive 404 maps to (nil, nil)
// (absence); any other non-OK code (403, 5xx) is returned as an error so the
// caller fails closed and never treats a permission/backend blip as a deletion.
func (i *PublicCloudVMInstanceClient) Read(ctx context.Context, id string) (*PublicCloudVMInstance, error) {
	req := i.c.newRequest("GET", "/vm_instances/v1/virtual_machines/%s", id)
	resp, err := i.c.doRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 404)
	if err != nil || !found {
		return nil, err
	}

	var out PublicCloudVMInstance
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateVMInstanceNIC is one network interface of the create body. The whole
// structure is immutable (ForceNew) on the resource; NIC changes go through the
// standalone network_adapter resource.
type CreateVMInstanceNIC struct {
	DeviceIndex int    `json:"deviceIndex"`
	NetworkID   string `json:"networkId"`
	IPAddress   string `json:"ipAddress,omitempty"`
}

// CreateVMInstanceCloudInit is the optional cloud-init payload of the create body.
type CreateVMInstanceCloudInit struct {
	CloudConfig   string `json:"cloudConfig,omitempty"`
	NetworkConfig string `json:"networkConfig,omitempty"`
}

// CreateVMInstanceRequest is the body of POST /vm_instances/v1/virtual_machines.
// disks[] is deliberately NOT modelled: the resource never creates disks (the
// template provides the system disk); data disks are the standalone disk
// resource. PowerState is passed through so the VM boots (or stays off) from the
// first apply; when empty the API leaves the VM stopped.
type CreateVMInstanceRequest struct {
	Name               string                     `json:"name"`
	AvailabilityZoneID string                     `json:"availabilityZoneId"`
	TemplateID         string                     `json:"templateId"`
	InstanceFamilyID   string                     `json:"instanceFamilyId"`
	CPU                int                        `json:"cpu"`
	Memory             int                        `json:"memory"`
	BackupPolicyID     string                     `json:"backupPolicyId"`
	NetworkInterfaces  []CreateVMInstanceNIC      `json:"networkInterfaces"`
	CloudInit          *CreateVMInstanceCloudInit `json:"cloudInit,omitempty"`
	PowerState         string                     `json:"powerState,omitempty"`
}

// Create submits the async VM creation and returns the raw activityId from the
// Location header. The caller waits on the activity and extracts the VM id from
// the completed activity (result / concernedItems type "vmi").
func (i *PublicCloudVMInstanceClient) Create(ctx context.Context, req *CreateVMInstanceRequest) (string, error) {
	r := i.c.newRequest("POST", "/vm_instances/v1/virtual_machines")
	r.obj = req
	return i.c.doRequestAndReturnActivity(ctx, r)
}

// PatchVMInstanceRequest is the body of PATCH /vm_instances/v1/virtual_machines/{id}
// (in-place metadata update). Both fields are pointers with omitempty so a
// diff-driven update sends only the attributes that actually changed.
type PatchVMInstanceRequest struct {
	Name           *string `json:"name,omitempty"`
	BackupPolicyID *string `json:"backupPolicyId,omitempty"`
}

// PatchMetadata submits the async in-place metadata update (name / backup policy)
// and returns the activityId.
func (i *PublicCloudVMInstanceClient) PatchMetadata(ctx context.Context, id string, req *PatchVMInstanceRequest) (string, error) {
	r := i.c.newRequest("PATCH", "/vm_instances/v1/virtual_machines/%s", id)
	r.obj = req
	return i.c.doRequestAndReturnActivity(ctx, r)
}

// ResizeVMInstanceRequest is the body of POST /vm_instances/v1/virtual_machines/{id}/resize.
// Both dimensions are optional (a resize may change only cpu or only memory).
// The "VM must be stopped" precondition is enforced downstream (worker); a
// violation surfaces as a failed activity.
type ResizeVMInstanceRequest struct {
	CPU    *int `json:"cpu,omitempty"`
	Memory *int `json:"memory,omitempty"`
}

// Resize submits the async vCPU/RAM resize and returns the activityId.
func (i *PublicCloudVMInstanceClient) Resize(ctx context.Context, id string, req *ResizeVMInstanceRequest) (string, error) {
	r := i.c.newRequest("POST", "/vm_instances/v1/virtual_machines/%s/resize", id)
	r.obj = req
	return i.c.doRequestAndReturnActivity(ctx, r)
}

// Start submits the async power-on transition and returns the activityId.
func (i *PublicCloudVMInstanceClient) Start(ctx context.Context, id string) (string, error) {
	r := i.c.newRequest("POST", "/vm_instances/v1/virtual_machines/%s/start", id)
	return i.c.doRequestAndReturnActivity(ctx, r)
}

// Stop submits the async power-off transition and returns the activityId.
func (i *PublicCloudVMInstanceClient) Stop(ctx context.Context, id string) (string, error) {
	r := i.c.newRequest("POST", "/vm_instances/v1/virtual_machines/%s/stop", id)
	return i.c.doRequestAndReturnActivity(ctx, r)
}

// Delete submits the async VM deletion and returns the activityId.
func (i *PublicCloudVMInstanceClient) Delete(ctx context.Context, id string) (string, error) {
	r := i.c.newRequest("DELETE", "/vm_instances/v1/virtual_machines/%s", id)
	return i.c.doRequestAndReturnActivity(ctx, r)
}
