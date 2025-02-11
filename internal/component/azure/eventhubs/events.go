/*
Copyright 2023 The Dapr Authors
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package eventhubs

import (
	"context"
	"strconv"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs"
	"github.com/spf13/cast"
)

// Type for the handler for messages coming in from the subscriptions.
type SubscribeHandler func(ctx context.Context, data []byte, metadata map[string]string) error

const (
	// Event Hubs SystemProperties names for metadata passthrough.
	sysPropSequenceNumber             = "x-opt-sequence-number"
	sysPropEnqueuedTime               = "x-opt-enqueued-time"
	sysPropOffset                     = "x-opt-offset"
	sysPropPartitionID                = "x-opt-partition-id"
	sysPropPartitionKey               = "x-opt-partition-key"
	sysPropIotHubDeviceConnectionID   = "iothub-connection-device-id"
	sysPropIotHubAuthGenerationID     = "iothub-connection-auth-generation-id"
	sysPropIotHubConnectionAuthMethod = "iothub-connection-auth-method"
	sysPropIotHubConnectionModuleID   = "iothub-connection-module-id"
	sysPropIotHubEnqueuedTime         = "iothub-enqueuedtime"
	sysPropMessageID                  = "message-id"
)

func subscribeHandler(ctx context.Context, getAllProperties bool, handler SubscribeHandler) func(e *azeventhubs.ReceivedEventData) error {
	return func(e *azeventhubs.ReceivedEventData) error {
		// Allocate with an initial capacity of 10 which covers the common properties, also from IoT Hub
		md := make(map[string]string, 10)

		md[sysPropSequenceNumber] = strconv.FormatInt(e.SequenceNumber, 10)
		if e.EnqueuedTime != nil {
			md[sysPropEnqueuedTime] = e.EnqueuedTime.Format(time.RFC3339)
		}
		if e.Offset != nil {
			md[sysPropOffset] = strconv.FormatInt(*e.Offset, 10)
		}
		if e.PartitionKey != nil {
			md[sysPropPartitionKey] = *e.PartitionKey
		}
		if e.MessageID != nil && *e.MessageID != "" {
			md[sysPropMessageID] = *e.MessageID
		}

		// Iterate through the system properties looking for those coming from IoT Hub
		for k, v := range e.SystemProperties {
			switch k {
			// The following metadata properties are only present if event was generated by Azure IoT Hub.
			case sysPropIotHubDeviceConnectionID,
				sysPropIotHubAuthGenerationID,
				sysPropIotHubConnectionAuthMethod,
				sysPropIotHubConnectionModuleID,
				sysPropIotHubEnqueuedTime:
				addPropertyToMetadata(k, v, md)
			default:
				// nop
			}
		}

		// Added properties if any (includes application properties from Azure IoT Hub)
		if getAllProperties && len(e.Properties) > 0 {
			for k, v := range e.Properties {
				addPropertyToMetadata(k, v, md)
			}
		}

		return handler(ctx, e.Body, md)
	}
}

// Adds a property to the response metadata
func addPropertyToMetadata(key string, value any, md map[string]string) {
	switch v := value.(type) {
	case *time.Time:
		if v != nil {
			md[key] = v.Format(time.RFC3339)
		}
	case time.Time:
		md[key] = v.Format(time.RFC3339)
	default:
		str, err := cast.ToStringE(value)
		if err == nil {
			md[key] = str
		}
	}
}
