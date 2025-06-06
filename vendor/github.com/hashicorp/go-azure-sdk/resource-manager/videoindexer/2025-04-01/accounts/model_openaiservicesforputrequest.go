package accounts

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See NOTICE.txt in the project root for license information.

type OpenAiServicesForPutRequest struct {
	ResourceId           *string `json:"resourceId,omitempty"`
	UserAssignedIdentity *string `json:"userAssignedIdentity,omitempty"`
}
