
## `github.com/hashicorp/go-azure-sdk/resource-manager/sqlvirtualmachine/2023-10-01/sqlvirtualmachines` Documentation

The `sqlvirtualmachines` SDK allows for interaction with Azure Resource Manager `sqlvirtualmachine` (API Version `2023-10-01`).

This readme covers example usages, but further information on [using this SDK can be found in the project root](https://github.com/hashicorp/go-azure-sdk/tree/main/docs).

### Import Path

```go
import "github.com/hashicorp/go-azure-helpers/resourcemanager/commonids"
import "github.com/hashicorp/go-azure-sdk/resource-manager/sqlvirtualmachine/2023-10-01/sqlvirtualmachines"
```


### Client Initialization

```go
client := sqlvirtualmachines.NewSqlVirtualMachinesClientWithBaseURI("https://management.azure.com")
client.Client.Authorizer = authorizer
```


### Example Usage: `SqlVirtualMachinesClient.CreateOrUpdate`

```go
ctx := context.TODO()
id := sqlvirtualmachines.NewSqlVirtualMachineID("12345678-1234-9876-4563-123456789012", "example-resource-group", "sqlVirtualMachineName")

payload := sqlvirtualmachines.SqlVirtualMachine{
	// ...
}


if err := client.CreateOrUpdateThenPoll(ctx, id, payload); err != nil {
	// handle the error
}
```


### Example Usage: `SqlVirtualMachinesClient.Delete`

```go
ctx := context.TODO()
id := sqlvirtualmachines.NewSqlVirtualMachineID("12345678-1234-9876-4563-123456789012", "example-resource-group", "sqlVirtualMachineName")

if err := client.DeleteThenPoll(ctx, id); err != nil {
	// handle the error
}
```


### Example Usage: `SqlVirtualMachinesClient.FetchDCAssessment`

```go
ctx := context.TODO()
id := sqlvirtualmachines.NewSqlVirtualMachineID("12345678-1234-9876-4563-123456789012", "example-resource-group", "sqlVirtualMachineName")

payload := sqlvirtualmachines.DiskConfigAssessmentRequest{
	// ...
}


if err := client.FetchDCAssessmentThenPoll(ctx, id, payload); err != nil {
	// handle the error
}
```


### Example Usage: `SqlVirtualMachinesClient.Get`

```go
ctx := context.TODO()
id := sqlvirtualmachines.NewSqlVirtualMachineID("12345678-1234-9876-4563-123456789012", "example-resource-group", "sqlVirtualMachineName")

read, err := client.Get(ctx, id, sqlvirtualmachines.DefaultGetOperationOptions())
if err != nil {
	// handle the error
}
if model := read.Model; model != nil {
	// do something with the model/response object
}
```


### Example Usage: `SqlVirtualMachinesClient.List`

```go
ctx := context.TODO()
id := commonids.NewSubscriptionID("12345678-1234-9876-4563-123456789012")

// alternatively `client.List(ctx, id)` can be used to do batched pagination
items, err := client.ListComplete(ctx, id)
if err != nil {
	// handle the error
}
for _, item := range items {
	// do something
}
```


### Example Usage: `SqlVirtualMachinesClient.ListByResourceGroup`

```go
ctx := context.TODO()
id := commonids.NewResourceGroupID("12345678-1234-9876-4563-123456789012", "example-resource-group")

// alternatively `client.ListByResourceGroup(ctx, id)` can be used to do batched pagination
items, err := client.ListByResourceGroupComplete(ctx, id)
if err != nil {
	// handle the error
}
for _, item := range items {
	// do something
}
```


### Example Usage: `SqlVirtualMachinesClient.Redeploy`

```go
ctx := context.TODO()
id := sqlvirtualmachines.NewSqlVirtualMachineID("12345678-1234-9876-4563-123456789012", "example-resource-group", "sqlVirtualMachineName")

if err := client.RedeployThenPoll(ctx, id); err != nil {
	// handle the error
}
```


### Example Usage: `SqlVirtualMachinesClient.SqlVirtualMachineTroubleshootTroubleshoot`

```go
ctx := context.TODO()
id := sqlvirtualmachines.NewSqlVirtualMachineID("12345678-1234-9876-4563-123456789012", "example-resource-group", "sqlVirtualMachineName")

payload := sqlvirtualmachines.SqlVMTroubleshooting{
	// ...
}


if err := client.SqlVirtualMachineTroubleshootTroubleshootThenPoll(ctx, id, payload); err != nil {
	// handle the error
}
```


### Example Usage: `SqlVirtualMachinesClient.StartAssessment`

```go
ctx := context.TODO()
id := sqlvirtualmachines.NewSqlVirtualMachineID("12345678-1234-9876-4563-123456789012", "example-resource-group", "sqlVirtualMachineName")

if err := client.StartAssessmentThenPoll(ctx, id); err != nil {
	// handle the error
}
```


### Example Usage: `SqlVirtualMachinesClient.Update`

```go
ctx := context.TODO()
id := sqlvirtualmachines.NewSqlVirtualMachineID("12345678-1234-9876-4563-123456789012", "example-resource-group", "sqlVirtualMachineName")

payload := sqlvirtualmachines.SqlVirtualMachineUpdate{
	// ...
}


if err := client.UpdateThenPoll(ctx, id, payload); err != nil {
	// handle the error
}
```
