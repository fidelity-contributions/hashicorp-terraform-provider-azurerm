---
subcategory: "Automation"
layout: "azurerm"
page_title: "Azure Resource Manager: azurerm_automation_connection_classic_certificate"
description: |-
  Manages an Automation Connection with type `AzureClassicCertificate`.
---

# azurerm_automation_connection_classic_certificate

Manages an Automation Connection with type `AzureClassicCertificate`.

## Example Usage

```hcl
provider "azurerm" {
  features {}
}

resource "azurerm_resource_group" "example" {
  name     = "resourceGroup-example"
  location = "West Europe"
}

data "azurerm_client_config" "example" {}

resource "azurerm_automation_account" "example" {
  name                = "account-example"
  location            = azurerm_resource_group.example.location
  resource_group_name = azurerm_resource_group.example.name
  sku_name            = "Basic"
}

resource "azurerm_automation_connection_classic_certificate" "example" {
  name                    = "connection-example"
  resource_group_name     = azurerm_resource_group.example.name
  automation_account_name = azurerm_automation_account.example.name
  certificate_asset_name  = "cert1"
  subscription_name       = "subs1"
  subscription_id         = data.azurerm_client_config.example.subscription_id
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) Specifies the name of the Connection. Changing this forces a new resource to be created.

* `resource_group_name` - (Required) The name of the resource group in which the Connection is created. Changing this forces a new resource to be created.

* `automation_account_name` - (Required) The name of the automation account in which the Connection is created. Changing this forces a new resource to be created.

* `certificate_asset_name` - (Required) The name of the certificate asset.

* `subscription_name` - (Required) The name of subscription.

* `subscription_id` - (Required) The id of subscription.

* `description` - (Optional) A description for this Connection.

## Attributes Reference

In addition to the Arguments listed above - the following Attributes are exported:

* `id` - The Automation Connection ID.

## Timeouts

The `timeouts` block allows you to specify [timeouts](https://www.terraform.io/language/resources/syntax#operation-timeouts) for certain actions:

* `create` - (Defaults to 30 minutes) Used when creating the Automation Connection.
* `read` - (Defaults to 5 minutes) Used when retrieving the Automation Connection.
* `update` - (Defaults to 30 minutes) Used when updating the Automation Connection.
* `delete` - (Defaults to 30 minutes) Used when deleting the Automation Connection.

## Import

Automation Connection can be imported using the `resource id`, e.g.

```shell
terraform import azurerm_automation_connection_classic_certificate.conn1 /subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/group1/providers/Microsoft.Automation/automationAccounts/account1/connections/conn1
```

## API Providers
<!-- This section is generated, changes will be overwritten -->
This resource uses the following Azure API Providers:

* `Microsoft.Automation` - 2023-11-01
