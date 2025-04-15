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
	TotalRuns     int
	SucessPercent float64
	Failed        int
	Warning       int
	Success       int
	Running       int
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
	FailedResources      int
	ProtectedResources   int
	UnprotectedResources int
	TotalResources       int
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
	InSPP               int
	InCompute           int
	WithBackup          int
	InSLA               int
	InOffloadingSLA     int
	TSMOffloadingFactor int
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
	Name                string
	TriggerType         string
	NumberOfProtectedVM int
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
	Version        string
	Build          string
	Date           string
	Product        string
	Epoch          int
	DeploymentType string
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
	CPUUtil int
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
