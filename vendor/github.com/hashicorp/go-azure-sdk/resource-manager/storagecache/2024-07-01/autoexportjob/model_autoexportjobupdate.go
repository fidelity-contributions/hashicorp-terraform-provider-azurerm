package autoexportjob

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See NOTICE.txt in the project root for license information.

type AutoExportJobUpdate struct {
	Properties *AutoExportJobUpdateProperties `json:"properties,omitempty"`
	Tags       *map[string]string             `json:"tags,omitempty"`
}
