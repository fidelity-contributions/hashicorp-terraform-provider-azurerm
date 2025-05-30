// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package monitor

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/go-azure-helpers/lang/pointer"
	"github.com/hashicorp/go-azure-helpers/lang/response"
	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonids"
	authRuleParse "github.com/hashicorp/go-azure-sdk/resource-manager/eventhub/2021-11-01/authorizationrulesnamespaces"
	"github.com/hashicorp/go-azure-sdk/resource-manager/insights/2021-05-01-preview/diagnosticsettings"
	"github.com/hashicorp/go-azure-sdk/resource-manager/operationalinsights/2020-08-01/workspaces"
	"github.com/hashicorp/terraform-provider-azurerm/helpers/azure"
	"github.com/hashicorp/terraform-provider-azurerm/helpers/tf"
	"github.com/hashicorp/terraform-provider-azurerm/internal/clients"
	"github.com/hashicorp/terraform-provider-azurerm/internal/features"
	eventhubValidate "github.com/hashicorp/terraform-provider-azurerm/internal/services/eventhub/validate"
	"github.com/hashicorp/terraform-provider-azurerm/internal/services/monitor/validate"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/pluginsdk"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/validation"
	"github.com/hashicorp/terraform-provider-azurerm/internal/timeouts"
)

func resourceMonitorDiagnosticSetting() *pluginsdk.Resource {
	resource := &pluginsdk.Resource{
		Create: resourceMonitorDiagnosticSettingCreate,
		Read:   resourceMonitorDiagnosticSettingRead,
		Update: resourceMonitorDiagnosticSettingUpdate,
		Delete: resourceMonitorDiagnosticSettingDelete,

		Importer: pluginsdk.ImporterValidatingResourceId(func(id string) error {
			_, err := ParseMonitorDiagnosticId(id)
			return err
		}),

		Timeouts: &pluginsdk.ResourceTimeout{
			Create: pluginsdk.DefaultTimeout(30 * time.Minute),
			Read:   pluginsdk.DefaultTimeout(5 * time.Minute),
			Update: pluginsdk.DefaultTimeout(30 * time.Minute),
			Delete: pluginsdk.DefaultTimeout(60 * time.Minute),
		},

		Schema: map[string]*pluginsdk.Schema{
			"name": {
				Type:         pluginsdk.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validate.MonitorDiagnosticSettingName,
			},

			"target_resource_id": {
				Type:         pluginsdk.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: azure.ValidateResourceID,
			},

			"eventhub_name": {
				Type:         pluginsdk.TypeString,
				Optional:     true,
				ValidateFunc: eventhubValidate.ValidateEventHubName(),
			},

			"eventhub_authorization_rule_id": {
				Type:         pluginsdk.TypeString,
				Optional:     true,
				ValidateFunc: authRuleParse.ValidateAuthorizationRuleID,
				AtLeastOneOf: []string{"eventhub_authorization_rule_id", "log_analytics_workspace_id", "storage_account_id", "partner_solution_id"},
			},

			"log_analytics_workspace_id": {
				Type:         pluginsdk.TypeString,
				Optional:     true,
				ValidateFunc: workspaces.ValidateWorkspaceID,
				AtLeastOneOf: []string{"eventhub_authorization_rule_id", "log_analytics_workspace_id", "storage_account_id", "partner_solution_id"},
			},

			"storage_account_id": {
				Type:         pluginsdk.TypeString,
				Optional:     true,
				ValidateFunc: commonids.ValidateStorageAccountID,
				AtLeastOneOf: []string{"eventhub_authorization_rule_id", "log_analytics_workspace_id", "storage_account_id", "partner_solution_id"},
			},

			"partner_solution_id": {
				Type:         pluginsdk.TypeString,
				Optional:     true,
				ValidateFunc: azure.ValidateResourceID,
				AtLeastOneOf: []string{"eventhub_authorization_rule_id", "log_analytics_workspace_id", "storage_account_id", "partner_solution_id"},
			},

			"log_analytics_destination_type": {
				Type:     pluginsdk.TypeString,
				Optional: true,
				ForceNew: false,
				Computed: true,
				ValidateFunc: validation.StringInSlice([]string{
					"Dedicated",
					"AzureDiagnostics", // Not documented in azure API, but some resource has skew. See: https://github.com/Azure/azure-rest-api-specs/issues/9281
				}, false),
			},

			"enabled_log": {
				Type:         pluginsdk.TypeSet,
				Optional:     true,
				AtLeastOneOf: []string{"enabled_log", "enabled_metric"},
				Elem: &pluginsdk.Resource{
					Schema: map[string]*pluginsdk.Schema{
						"category": {
							Type:         pluginsdk.TypeString,
							Optional:     true,
							ValidateFunc: validation.StringIsNotEmpty,
						},

						"category_group": {
							Type:         pluginsdk.TypeString,
							Optional:     true,
							ValidateFunc: validation.StringIsNotEmpty,
						},

						"retention_policy": {
							Type:       pluginsdk.TypeList,
							Optional:   true,
							MaxItems:   1,
							Deprecated: "`retention_policy` has been deprecated in favor of `azurerm_storage_management_policy` resource - to learn more https://aka.ms/diagnostic_settings_log_retention",
							Elem: &pluginsdk.Resource{
								Schema: map[string]*pluginsdk.Schema{
									"enabled": {
										Type:     pluginsdk.TypeBool,
										Required: true,
									},

									"days": {
										Type:         pluginsdk.TypeInt,
										Optional:     true,
										ValidateFunc: validation.IntAtLeast(0),
									},
								},
							},
						},
					},
				},
				Set: resourceMonitorDiagnosticLogSettingHash,
			},

			"enabled_metric": {
				Type:         pluginsdk.TypeSet,
				Optional:     true,
				AtLeastOneOf: []string{"enabled_log", "enabled_metric"},
				Elem: &pluginsdk.Resource{
					Schema: map[string]*pluginsdk.Schema{
						"category": {
							Type:         pluginsdk.TypeString,
							Required:     true,
							ValidateFunc: validation.StringIsNotEmpty,
						},
					},
				},
			},
		},
	}

	if !features.FivePointOh() {
		resource.Schema["enabled_log"].AtLeastOneOf = []string{"enabled_log", "enabled_metric", "metric"}

		resource.Schema["enabled_metric"].AtLeastOneOf = []string{"enabled_log", "enabled_metric", "metric"}
		resource.Schema["enabled_metric"].Computed = true
		resource.Schema["enabled_metric"].ConflictsWith = []string{"metric"}

		resource.Schema["metric"] = &pluginsdk.Schema{
			Type:          pluginsdk.TypeSet,
			Optional:      true,
			Computed:      true,
			Deprecated:    "`metric` has been deprecated in favor of the `enabled_metric` property and will be removed in v5.0 of the AzureRM provider",
			AtLeastOneOf:  []string{"enabled_log", "enabled_metric", "metric"},
			ConflictsWith: []string{"enabled_metric"},
			Set:           resourceMonitorDiagnosticMetricsSettingHash,
			Elem: &pluginsdk.Resource{
				Schema: map[string]*pluginsdk.Schema{
					"category": {
						Type:         pluginsdk.TypeString,
						Required:     true,
						ValidateFunc: validation.StringIsNotEmpty,
					},

					"enabled": {
						Type:     pluginsdk.TypeBool,
						Optional: true,
						Default:  true,
					},

					"retention_policy": {
						Type:       pluginsdk.TypeList,
						Optional:   true,
						MaxItems:   1,
						Deprecated: "`retention_policy` has been deprecated in favor of the `azurerm_storage_management_policy` resource and will be removed in v5.0 of the AzureRM provider - to learn more go to https://aka.ms/diagnostic_settings_log_retention",
						Elem: &pluginsdk.Resource{
							Schema: map[string]*pluginsdk.Schema{
								"enabled": {
									Type:     pluginsdk.TypeBool,
									Required: true,
								},

								"days": {
									Type:         pluginsdk.TypeInt,
									Optional:     true,
									ValidateFunc: validation.IntAtLeast(0),
								},
							},
						},
					},
				},
			},
		}
	}

	return resource
}

func resourceMonitorDiagnosticSettingCreate(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Monitor.DiagnosticSettingsClient
	ctx, cancel := timeouts.ForCreate(meta.(*clients.Client).StopContext, d)
	defer cancel()
	log.Printf("[INFO] preparing arguments for Azure ARM Diagnostic Settings.")

	id := diagnosticsettings.NewScopedDiagnosticSettingID(d.Get("target_resource_id").(string), d.Get("name").(string))
	resourceId := fmt.Sprintf("%s|%s", id.ResourceUri, id.DiagnosticSettingName)

	existing, err := client.Get(ctx, id)
	if err != nil {
		if !response.WasNotFound(existing.HttpResponse) {
			return fmt.Errorf("checking for presence of existing Monitor Diagnostic Setting %q for Resource %q: %s", id.DiagnosticSettingName, id.ResourceUri, err)
		}
	}

	if !response.WasNotFound(existing.HttpResponse) {
		return tf.ImportAsExistsError("azurerm_monitor_diagnostic_setting", resourceId)
	}

	var logs []diagnosticsettings.LogSettings
	hasEnabledLogs := false
	if enabledLogs, ok := d.GetOk("enabled_log"); ok {
		enabledLogsList := enabledLogs.(*pluginsdk.Set).List()
		if len(enabledLogsList) > 0 {
			expandEnabledLogs, err := expandMonitorDiagnosticsSettingsEnabledLogs(enabledLogsList)
			if err != nil {
				return fmt.Errorf("expanding enabled_log: %+v", err)
			}
			logs = *expandEnabledLogs
			hasEnabledLogs = true
		}
	}

	// if no logs/metrics are enabled the API "creates" but 404's on Read
	var metrics []diagnosticsettings.MetricSettings
	hasEnabledMetrics := false

	if !features.FivePointOh() {
		metrics = expandMonitorDiagnosticsSettingsMetrics(d.Get("metric").(*pluginsdk.Set).List())
		for _, v := range metrics {
			if v.Enabled {
				hasEnabledMetrics = true
				break
			}
		}
	}

	if enabledMetrics, ok := d.GetOk("enabled_metric"); ok {
		enabledMetricsList := enabledMetrics.(*pluginsdk.Set).List()
		if len(enabledMetricsList) > 0 {
			metrics = expandMonitorDiagnosticsSettingsEnabledMetrics(enabledMetricsList)
			hasEnabledMetrics = true
		}
	}

	if !hasEnabledMetrics && !hasEnabledLogs {
		return fmt.Errorf("at least one type of Log or Metric must be enabled")
	}

	parameters := diagnosticsettings.DiagnosticSettingsResource{
		Properties: &diagnosticsettings.DiagnosticSettings{
			Logs:    &logs,
			Metrics: &metrics,
		},
	}

	eventHubAuthorizationRuleId := d.Get("eventhub_authorization_rule_id").(string)
	eventHubName := d.Get("eventhub_name").(string)
	if eventHubAuthorizationRuleId != "" {
		parameters.Properties.EventHubAuthorizationRuleId = pointer.To(eventHubAuthorizationRuleId)
		parameters.Properties.EventHubName = pointer.To(eventHubName)
	}

	workspaceId := d.Get("log_analytics_workspace_id").(string)
	if workspaceId != "" {
		parameters.Properties.WorkspaceId = pointer.To(workspaceId)
	}

	storageAccountId := d.Get("storage_account_id").(string)
	if storageAccountId != "" {
		parameters.Properties.StorageAccountId = pointer.To(storageAccountId)
	}

	partnerSolutionId := d.Get("partner_solution_id").(string)
	if partnerSolutionId != "" {
		parameters.Properties.MarketplacePartnerId = pointer.To(partnerSolutionId)
	}

	if v := d.Get("log_analytics_destination_type").(string); v != "" {
		parameters.Properties.LogAnalyticsDestinationType = &v
	}

	if _, err := client.CreateOrUpdate(ctx, id, parameters); err != nil {
		return fmt.Errorf("creating Monitor Diagnostics Setting %q for Resource %q: %+v", id.DiagnosticSettingName, id.ResourceUri, err)
	}

	deadline, ok := ctx.Deadline()
	if !ok {
		return fmt.Errorf("internal error: could not retrieve context deadline for %s", id.ID())
	}

	// https://github.com/Azure/azure-rest-api-specs/issues/30249
	log.Printf("[DEBUG] Waiting for Monitor Diagnostic Setting %q for Resource %q to become ready", id.DiagnosticSettingName, id.ResourceUri)
	stateConf := &pluginsdk.StateChangeConf{
		Pending:                   []string{"NotFound"},
		Target:                    []string{"Exists"},
		Refresh:                   monitorDiagnosticSettingRefreshFunc(ctx, client, id),
		MinTimeout:                5 * time.Second,
		ContinuousTargetOccurence: 3,
		Timeout:                   time.Until(deadline),
	}

	if _, err = stateConf.WaitForStateContext(ctx); err != nil {
		return fmt.Errorf("waiting for Monitor Diagnostic Setting %q for Resource %q to become ready: %s", id.DiagnosticSettingName, id.ResourceUri, err)
	}

	d.SetId(resourceId)

	return resourceMonitorDiagnosticSettingRead(d, meta)
}

func resourceMonitorDiagnosticSettingUpdate(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Monitor.DiagnosticSettingsClient
	ctx, cancel := timeouts.ForUpdate(meta.(*clients.Client).StopContext, d)
	defer cancel()
	log.Printf("[INFO] preparing arguments for Azure ARM Diagnostic Settings.")

	id, err := ParseMonitorDiagnosticId(d.Id())
	if err != nil {
		return err
	}

	existing, err := client.Get(ctx, *id)
	if err != nil {
		return fmt.Errorf("retrieving Monitor Diagnostics Setting %q for Resource %q: %+v", id.DiagnosticSettingName, id.ResourceUri, err)
	}
	if existing.Model == nil || existing.Model.Properties == nil {
		return fmt.Errorf("unexpected null model of Monitor Diagnostics Setting %q for Resource %q", id.DiagnosticSettingName, id.ResourceUri)
	}

	var logs []diagnosticsettings.LogSettings
	hasEnabledLogs := false

	if d.HasChange("enabled_log") {
		enabledLogs := d.Get("enabled_log").(*pluginsdk.Set).List()
		log.Printf("[DEBUG] enabled_logs: %+v", enabledLogs)
		if len(enabledLogs) > 0 {
			expandEnabledLogs, err := expandMonitorDiagnosticsSettingsEnabledLogs(enabledLogs)
			if err != nil {
				return fmt.Errorf("expanding enabled_log: %+v", err)
			}
			logs = *expandEnabledLogs
			hasEnabledLogs = true
		} else if existing.Model != nil && existing.Model.Properties != nil && existing.Model.Properties.Logs != nil {
			// if the enabled_log is updated to empty, we disable the log explicitly
			for _, v := range *existing.Model.Properties.Logs {
				disabledLog := v
				disabledLog.Enabled = false
				logs = append(logs, disabledLog)
			}
		}
	} else if existing.Model != nil && existing.Model.Properties != nil && existing.Model.Properties.Logs != nil {
		logs = *existing.Model.Properties.Logs
		for _, v := range logs {
			if v.Enabled {
				hasEnabledLogs = true
			}
		}
	}

	var metrics []diagnosticsettings.MetricSettings
	hasEnabledMetrics := false

	if d.HasChange("enabled_metric") {
		enabledMetrics := d.Get("enabled_metric").(*pluginsdk.Set).List()
		if len(enabledMetrics) > 0 {
			expandEnabledMetrics := expandMonitorDiagnosticsSettingsEnabledMetrics(enabledMetrics)
			metrics = expandEnabledMetrics
			hasEnabledMetrics = true
		} else if existing.Model != nil && existing.Model.Properties != nil && existing.Model.Properties.Metrics != nil {
			// if the enabled_metric is updated to empty, we disable the metric explicitly
			for _, v := range *existing.Model.Properties.Metrics {
				disabledMetric := v
				disabledMetric.Enabled = false
				metrics = append(metrics, disabledMetric)
			}
		}
	} else if existing.Model != nil && existing.Model.Properties != nil && existing.Model.Properties.Metrics != nil {
		metrics = *existing.Model.Properties.Metrics
		for _, v := range metrics {
			if v.Enabled {
				hasEnabledMetrics = true
			}
		}
	}

	if !features.FivePointOh() {
		if d.HasChange("metric") {
			metrics = expandMonitorDiagnosticsSettingsMetrics(d.Get("metric").(*pluginsdk.Set).List())
			for _, v := range metrics {
				if v.Enabled {
					hasEnabledMetrics = true
					break
				}
			}
		}
	}

	// if no logs/metrics are enabled the API "creates" but 404's on Read
	if !hasEnabledMetrics && !hasEnabledLogs {
		return fmt.Errorf("at least one type of Log or Metric must be enabled")
	}

	parameters := diagnosticsettings.DiagnosticSettingsResource{
		Properties: &diagnosticsettings.DiagnosticSettings{
			Logs:    &logs,
			Metrics: &metrics,
		},
	}

	eventHubAuthorizationRuleId := d.Get("eventhub_authorization_rule_id").(string)
	eventHubName := d.Get("eventhub_name").(string)
	if eventHubAuthorizationRuleId != "" {
		parameters.Properties.EventHubAuthorizationRuleId = pointer.To(eventHubAuthorizationRuleId)
		parameters.Properties.EventHubName = pointer.To(eventHubName)
	}

	workspaceId := d.Get("log_analytics_workspace_id").(string)
	if workspaceId != "" {
		parameters.Properties.WorkspaceId = pointer.To(workspaceId)
	}

	storageAccountId := d.Get("storage_account_id").(string)
	if storageAccountId != "" {
		parameters.Properties.StorageAccountId = pointer.To(storageAccountId)
	}

	partnerSolutionId := d.Get("partner_solution_id").(string)
	if partnerSolutionId != "" {
		parameters.Properties.MarketplacePartnerId = pointer.To(partnerSolutionId)
	}

	if v := d.Get("log_analytics_destination_type").(string); v != "" {
		parameters.Properties.LogAnalyticsDestinationType = &v
	}

	if _, err := client.CreateOrUpdate(ctx, *id, parameters); err != nil {
		return fmt.Errorf("updating Monitor Diagnostics Setting %q for Resource %q: %+v", id.DiagnosticSettingName, id.ResourceUri, err)
	}
	return resourceMonitorDiagnosticSettingRead(d, meta)
}

func resourceMonitorDiagnosticSettingRead(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Monitor.DiagnosticSettingsClient
	ctx, cancel := timeouts.ForRead(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := ParseMonitorDiagnosticId(d.Id())
	if err != nil {
		return err
	}

	resp, err := client.Get(ctx, *id)
	if err != nil {
		if response.WasNotFound(resp.HttpResponse) {
			log.Printf("[WARN] Monitor Diagnostics Setting %q was not found for Resource %q - removing from state!", id.DiagnosticSettingName, id.ResourceUri)
			d.SetId("")
			return nil
		}

		return fmt.Errorf("retrieving Monitor Diagnostics Setting %q for Resource %q: %+v", id.DiagnosticSettingName, id.ResourceUri, err)
	}

	d.Set("name", id.DiagnosticSettingName)
	resourceUri := id.ResourceUri
	if v, err := commonids.ParseKustoClusterIDInsensitively(resourceUri); err == nil {
		resourceUri = v.ID()
	}
	d.Set("target_resource_id", resourceUri)

	if model := resp.Model; model != nil {
		if props := model.Properties; props != nil {
			d.Set("eventhub_name", props.EventHubName)
			eventhubAuthorizationRuleId := ""
			if props.EventHubAuthorizationRuleId != nil && *props.EventHubAuthorizationRuleId != "" {
				authRuleId := pointer.From(props.EventHubAuthorizationRuleId)
				parsedId, err := authRuleParse.ParseAuthorizationRuleIDInsensitively(authRuleId)
				if err != nil {
					return err
				}
				eventhubAuthorizationRuleId = parsedId.ID()
			}
			d.Set("eventhub_authorization_rule_id", eventhubAuthorizationRuleId)

			workspaceId := ""
			if props.WorkspaceId != nil && *props.WorkspaceId != "" {
				parsedId, err := workspaces.ParseWorkspaceIDInsensitively(*props.WorkspaceId)
				if err != nil {
					return err
				}

				workspaceId = parsedId.ID()
			}
			d.Set("log_analytics_workspace_id", workspaceId)

			if props.StorageAccountId != nil && *props.StorageAccountId != "" {
				parsedId, err := commonids.ParseStorageAccountIDInsensitively(*props.StorageAccountId)
				if err != nil {
					return err
				}

				d.Set("storage_account_id", parsedId.ID())
			}

			if props.MarketplacePartnerId != nil && *props.MarketplacePartnerId != "" {
				d.Set("partner_solution_id", props.MarketplacePartnerId)
			}

			logAnalyticsDestinationType := ""
			if resp.Model.Properties.LogAnalyticsDestinationType != nil && *resp.Model.Properties.LogAnalyticsDestinationType != "" {
				logAnalyticsDestinationType = *resp.Model.Properties.LogAnalyticsDestinationType
			}
			d.Set("log_analytics_destination_type", logAnalyticsDestinationType)

			enabledLogs := flattenMonitorDiagnosticEnabledLogs(resp.Model.Properties.Logs)
			if err = d.Set("enabled_log", enabledLogs); err != nil {
				return fmt.Errorf("setting `enabled_log`: %+v", err)
			}

			if err = d.Set("enabled_metric", flattenMonitorDiagnosticEnabledMetrics(resp.Model.Properties.Metrics)); err != nil {
				return fmt.Errorf("setting `enabled_metric`: %+v", err)
			}

			if !features.FivePointOh() {
				if err := d.Set("metric", flattenMonitorDiagnosticMetrics(resp.Model.Properties.Metrics)); err != nil {
					return fmt.Errorf("setting `metric`: %+v", err)
				}
			}
		}
	}

	return nil
}

func resourceMonitorDiagnosticSettingDelete(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Monitor.DiagnosticSettingsClient
	ctx, cancel := timeouts.ForDelete(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := ParseMonitorDiagnosticId(d.Id())
	if err != nil {
		return err
	}

	resp, err := client.Delete(ctx, *id)
	if err != nil {
		if !response.WasNotFound(resp.HttpResponse) {
			return fmt.Errorf("deleting Monitor Diagnostics Setting %q for Resource %q: %+v", id.DiagnosticSettingName, id.ResourceUri, err)
		}
	}

	deadline, ok := ctx.Deadline()
	if !ok {
		return fmt.Errorf("internal error: could not retrieve context deadline for %s", id.ID())
	}

	// API appears to be eventually consistent (identified during tainting this resource)
	log.Printf("[DEBUG] Waiting for Monitor Diagnostic Setting %q for Resource %q to disappear", id.DiagnosticSettingName, id.ResourceUri)
	stateConf := &pluginsdk.StateChangeConf{
		Pending:                   []string{"Exists"},
		Target:                    []string{"NotFound"},
		Refresh:                   monitorDiagnosticSettingRefreshFunc(ctx, client, *id),
		MinTimeout:                15 * time.Second,
		ContinuousTargetOccurence: 5,
		Timeout:                   time.Until(deadline),
	}

	if _, err = stateConf.WaitForStateContext(ctx); err != nil {
		return fmt.Errorf("waiting for Monitor Diagnostic Setting %q for Resource %q to disappear: %s", id.DiagnosticSettingName, id.ResourceUri, err)
	}

	return nil
}

func monitorDiagnosticSettingRefreshFunc(ctx context.Context, client *diagnosticsettings.DiagnosticSettingsClient, targetResourceId diagnosticsettings.ScopedDiagnosticSettingId) pluginsdk.StateRefreshFunc {
	return func() (interface{}, string, error) {
		res, err := client.Get(ctx, targetResourceId)
		if err != nil {
			if response.WasNotFound(res.HttpResponse) {
				return "NotFound", "NotFound", nil
			}
			return nil, "", fmt.Errorf("issuing read request in monitorDiagnosticSettingRefreshFunc: %s", err)
		}

		return res, "Exists", nil
	}
}

func expandMonitorDiagnosticsSettingsEnabledLogs(input []interface{}) (*[]diagnosticsettings.LogSettings, error) {
	results := make([]diagnosticsettings.LogSettings, 0)

	for _, raw := range input {
		v := raw.(map[string]interface{})

		category := v["category"].(string)
		categoryGroup := v["category_group"].(string)
		policiesRaw := v["retention_policy"].([]interface{})
		var retentionPolicy *diagnosticsettings.RetentionPolicy
		if len(policiesRaw) != 0 {
			policyRaw := policiesRaw[0].(map[string]interface{})
			retentionDays := policyRaw["days"].(int)
			retentionEnabled := policyRaw["enabled"].(bool)
			retentionPolicy = &diagnosticsettings.RetentionPolicy{
				Days:    int64(retentionDays),
				Enabled: retentionEnabled,
			}
		}

		output := diagnosticsettings.LogSettings{
			Enabled:         true,
			RetentionPolicy: retentionPolicy,
		}

		switch {
		case category != "":
			output.Category = pointer.To(category)
		case categoryGroup != "":
			output.CategoryGroup = pointer.To(categoryGroup)
		default:
			return nil, fmt.Errorf("exactly one of `category` or `category_group` must be specified")
		}

		results = append(results, output)
	}

	return &results, nil
}

func flattenMonitorDiagnosticEnabledLogs(input *[]diagnosticsettings.LogSettings) []interface{} {
	enabledLogs := make([]interface{}, 0)
	if input == nil {
		return enabledLogs
	}

	for _, v := range *input {
		output := make(map[string]interface{})

		if !v.Enabled {
			continue
		}

		category := ""
		if v.Category != nil {
			category = *v.Category
		}
		output["category"] = category

		categoryGroup := ""
		if v.CategoryGroup != nil {
			categoryGroup = *v.CategoryGroup
		}
		output["category_group"] = categoryGroup

		policies := make([]interface{}, 0)

		if inputPolicy := v.RetentionPolicy; inputPolicy != nil {
			outputPolicy := make(map[string]interface{})

			outputPolicy["days"] = int(inputPolicy.Days)

			outputPolicy["enabled"] = inputPolicy.Enabled

			policies = append(policies, outputPolicy)
		}

		output["retention_policy"] = policies

		enabledLogs = append(enabledLogs, output)
	}
	return enabledLogs
}

func flattenMonitorDiagnosticEnabledMetrics(input *[]diagnosticsettings.MetricSettings) []interface{} {
	enabledLogs := make([]interface{}, 0)
	if input == nil {
		return enabledLogs
	}

	for _, v := range *input {
		output := make(map[string]interface{})

		if !v.Enabled {
			continue
		}

		output["category"] = pointer.From(v.Category)

		enabledLogs = append(enabledLogs, output)
	}
	return enabledLogs
}

func expandMonitorDiagnosticsSettingsMetrics(input []interface{}) []diagnosticsettings.MetricSettings {
	results := make([]diagnosticsettings.MetricSettings, 0)

	for _, raw := range input {
		v := raw.(map[string]interface{})

		output := diagnosticsettings.MetricSettings{
			Category: pointer.To(v["category"].(string)),
			Enabled:  v["enabled"].(bool),
		}

		if !features.FivePointOh() {
			policiesRaw := v["retention_policy"].([]interface{})
			var retentionPolicy *diagnosticsettings.RetentionPolicy
			if len(policiesRaw) > 0 && policiesRaw[0] != nil {
				policyRaw := policiesRaw[0].(map[string]interface{})
				retentionDays := policyRaw["days"].(int)
				retentionEnabled := policyRaw["enabled"].(bool)
				retentionPolicy = &diagnosticsettings.RetentionPolicy{
					Days:    int64(retentionDays),
					Enabled: retentionEnabled,
				}
			}
			output.RetentionPolicy = retentionPolicy
		}

		results = append(results, output)
	}

	return results
}

func expandMonitorDiagnosticsSettingsEnabledMetrics(input []interface{}) []diagnosticsettings.MetricSettings {
	results := make([]diagnosticsettings.MetricSettings, 0)

	for _, raw := range input {
		v := raw.(map[string]interface{})

		output := diagnosticsettings.MetricSettings{
			Category: pointer.To(v["category"].(string)),
			Enabled:  true,
		}

		results = append(results, output)
	}

	return results
}

func flattenMonitorDiagnosticMetrics(input *[]diagnosticsettings.MetricSettings) []interface{} {
	results := make([]interface{}, 0)
	if input == nil {
		return results
	}

	for _, v := range *input {
		output := make(map[string]interface{})

		if v.Category != nil {
			output["category"] = *v.Category
		}

		output["enabled"] = v.Enabled

		if !features.FivePointOh() {
			policies := make([]interface{}, 0)

			if inputPolicy := v.RetentionPolicy; inputPolicy != nil {
				outputPolicy := make(map[string]interface{})

				outputPolicy["days"] = int(inputPolicy.Days)

				outputPolicy["enabled"] = inputPolicy.Enabled

				policies = append(policies, outputPolicy)
			}
			output["retention_policy"] = policies
		}

		results = append(results, output)
	}

	return results
}

func ParseMonitorDiagnosticId(monitorId string) (*diagnosticsettings.ScopedDiagnosticSettingId, error) {
	v := strings.Split(monitorId, "|")
	if len(v) != 2 {
		return nil, fmt.Errorf("expected the Monitor Diagnostics ID to be in the format `{resourceId}|{name}` but got %d segments", len(v))
	}

	// TODO: this can become a Composite Resource ID once https://github.com/hashicorp/go-azure-helpers/pull/208 is released
	identifier := diagnosticsettings.NewScopedDiagnosticSettingID(v[0], v[1])
	return &identifier, nil
}

func resourceMonitorDiagnosticLogSettingHash(input interface{}) int {
	var buf bytes.Buffer
	if rawData, ok := input.(map[string]interface{}); ok {
		if category, ok := rawData["category"]; ok {
			buf.WriteString(fmt.Sprintf("%s-", category.(string)))
		}
		if categoryGroup, ok := rawData["category_group"]; ok {
			buf.WriteString(fmt.Sprintf("%s-", categoryGroup.(string)))
		}
		if enabled, ok := rawData["enabled"]; ok {
			buf.WriteString(fmt.Sprintf("%t-", enabled.(bool)))
		}
		if policy, ok := rawData["retention_policy"].(map[string]interface{}); ok {
			if policyEnabled, ok := policy["enabled"]; ok {
				buf.WriteString(fmt.Sprintf("%t-", policyEnabled.(bool)))
			}
			if days, ok := policy["days"]; ok {
				buf.WriteString(fmt.Sprintf("%d-", days.(int)))
			}
		}
	}
	return pluginsdk.HashString(buf.String())
}

func resourceMonitorDiagnosticMetricsSettingHash(input interface{}) int {
	var buf bytes.Buffer
	if rawData, ok := input.(map[string]interface{}); ok {
		if category, ok := rawData["category"]; ok {
			buf.WriteString(fmt.Sprintf("%s-", category.(string)))
		}
		if enabled, ok := rawData["enabled"]; ok {
			buf.WriteString(fmt.Sprintf("%t-", enabled.(bool)))
		}
		if policy, ok := rawData["retention_policy"].(map[string]interface{}); ok {
			if policyEnabled, ok := policy["enabled"]; ok {
				buf.WriteString(fmt.Sprintf("%t-", policyEnabled.(bool)))
			}
			if days, ok := policy["days"]; ok {
				buf.WriteString(fmt.Sprintf("%d-", days.(int)))
			}
		}
	}
	return pluginsdk.HashString(buf.String())
}
