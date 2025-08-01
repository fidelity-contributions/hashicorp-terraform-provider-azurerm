package privatezones

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See NOTICE.txt in the project root for license information.

type SoaRecord struct {
	Email        *string `json:"email,omitempty"`
	ExpireTime   *int64  `json:"expireTime,omitempty"`
	Host         *string `json:"host,omitempty"`
	MinimumTtl   *int64  `json:"minimumTtl,omitempty"`
	RefreshTime  *int64  `json:"refreshTime,omitempty"`
	RetryTime    *int64  `json:"retryTime,omitempty"`
	SerialNumber *int64  `json:"serialNumber,omitempty"`
}
