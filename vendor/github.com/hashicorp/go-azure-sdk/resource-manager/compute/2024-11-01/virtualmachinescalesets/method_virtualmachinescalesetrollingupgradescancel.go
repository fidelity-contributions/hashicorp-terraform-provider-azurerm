package virtualmachinescalesets

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/go-azure-sdk/sdk/client"
	"github.com/hashicorp/go-azure-sdk/sdk/client/pollers"
	"github.com/hashicorp/go-azure-sdk/sdk/client/resourcemanager"
	"github.com/hashicorp/go-azure-sdk/sdk/odata"
)

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See NOTICE.txt in the project root for license information.

type VirtualMachineScaleSetRollingUpgradesCancelOperationResponse struct {
	Poller       pollers.Poller
	HttpResponse *http.Response
	OData        *odata.OData
}

// VirtualMachineScaleSetRollingUpgradesCancel ...
func (c VirtualMachineScaleSetsClient) VirtualMachineScaleSetRollingUpgradesCancel(ctx context.Context, id VirtualMachineScaleSetId) (result VirtualMachineScaleSetRollingUpgradesCancelOperationResponse, err error) {
	opts := client.RequestOptions{
		ContentType: "application/json; charset=utf-8",
		ExpectedStatusCodes: []int{
			http.StatusAccepted,
			http.StatusOK,
		},
		HttpMethod: http.MethodPost,
		Path:       fmt.Sprintf("%s/rollingUpgrades/cancel", id.ID()),
	}

	req, err := c.Client.NewRequest(ctx, opts)
	if err != nil {
		return
	}

	var resp *client.Response
	resp, err = req.Execute(ctx)
	if resp != nil {
		result.OData = resp.OData
		result.HttpResponse = resp.Response
	}
	if err != nil {
		return
	}

	result.Poller, err = resourcemanager.PollerFromResponse(resp, c.Client)
	if err != nil {
		return
	}

	return
}

// VirtualMachineScaleSetRollingUpgradesCancelThenPoll performs VirtualMachineScaleSetRollingUpgradesCancel then polls until it's completed
func (c VirtualMachineScaleSetsClient) VirtualMachineScaleSetRollingUpgradesCancelThenPoll(ctx context.Context, id VirtualMachineScaleSetId) error {
	result, err := c.VirtualMachineScaleSetRollingUpgradesCancel(ctx, id)
	if err != nil {
		return fmt.Errorf("performing VirtualMachineScaleSetRollingUpgradesCancel: %+v", err)
	}

	if err := result.Poller.PollUntilDone(ctx); err != nil {
		return fmt.Errorf("polling after VirtualMachineScaleSetRollingUpgradesCancel: %+v", err)
	}

	return nil
}
