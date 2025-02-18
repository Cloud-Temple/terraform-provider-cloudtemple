package client

import (
	"context"
	"strconv"
)

type BackupMetricsClient struct {
	c *Client
}

func (c *BackupClient) Metrics() *BackupMetricsClient {
	return &BackupMetricsClient{c.c}
}

type BackupMetricsHistory struct {
	TotalRuns     int     `terraform:"total_runs"`
	SucessPercent float64 `terraform:"sucess_percent"`
	Failed        int     `terraform:"failed"`
	Warning       int     `terraform:"warning"`
	Success       int     `terraform:"success"`
	Running       int     `terraform:"running"`
}

func (c *BackupMetricsClient) History(ctx context.Context, rang int) (*BackupMetricsHistory, error) {
	r := c.c.newRequest("GET", "/backup/v1/spp/metrics/backup/history")
	r.params.Add("range", strconv.Itoa(rang))
	resp, err := c.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 403)
	if err != nil || !found {
		return nil, err
	}

	var out BackupMetricsHistory
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

type BackupMetricsCoverage struct {
	FailedResources      int `terraform:"failed_resources"`
	ProtectedResources   int `terraform:"protected_resources"`
	UnprotectedResources int `terraform:"unprotected_resources"`
	TotalResources       int `terraform:"total_resources"`
}

func (c *BackupMetricsClient) Coverage(ctx context.Context) (*BackupMetricsCoverage, error) {
	r := c.c.newRequest("GET", "/backup/v1/spp/metrics/coverage")
	resp, err := c.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out BackupMetricsCoverage
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

type BackupMetricsVirtualMachines struct {
	InSPP               int `terraform:"in_spp"`
	InCompute           int `terraform:"in_compute"`
	WithBackup          int `terraform:"with_backup"`
	InSLA               int `terraform:"in_sla"`
	InOffloadingSLA     int `terraform:"in_offloading_sla"`
	TSMOffloadingFactor int `terraform:"tsm_offloading_factor"`
}

func (c *BackupMetricsClient) VirtualMachines(ctx context.Context) (*BackupMetricsVirtualMachines, error) {
	r := c.c.newRequest("GET", "/backup/v1/spp/metrics/vm")
	resp, err := c.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out BackupMetricsVirtualMachines
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

type BackupMetricsPolicies struct {
	Name                string `terraform:"name"`
	TriggerType         string `terraform:"trigger_type"`
	NumberOfProtectedVM int    `terraform:"number_of_protected_vm"`
}

func (c *BackupMetricsClient) Policies(ctx context.Context) ([]*BackupMetricsPolicies, error) {
	r := c.c.newRequest("GET", "/backup/v1/spp/metrics/policies")
	resp, err := c.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out []*BackupMetricsPolicies
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}

type BackupMetricsPlatform struct {
	Version        string `terraform:"version"`
	Build          string `terraform:"build"`
	Date           string `terraform:"date"`
	Product        string `terraform:"product"`
	Epoch          int    `terraform:"epoch"`
	DeploymentType string `terraform:"deployment_type"`
}

func (c *BackupMetricsClient) Platform(ctx context.Context) (*BackupMetricsPlatform, error) {
	r := c.c.newRequest("GET", "/backup/v1/spp/metrics/plateform")
	resp, err := c.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out *BackupMetricsPlatform
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}

type BackupMetricsPlatformCPU struct {
	CPUUtil int `terraform:"cpu_util"`
}

func (c *BackupMetricsClient) PlatformCPU(ctx context.Context) (*BackupMetricsPlatformCPU, error) {
	r := c.c.newRequest("GET", "/backup/v1/spp/metrics/plateform/cpu")
	resp, err := c.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 403)
	if err != nil || !found {
		return nil, err
	}

	var out BackupMetricsPlatformCPU
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}
