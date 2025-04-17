// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"fmt"
	_ "unsafe"
)

//go:linkname AwsConfigHook main.AwsConfigHook
func AwsConfigHook(cfg interface{}) {
	fmt.Printf("AWS Config Hook: Auto-instrumenting AWS SDK calls\n")
}

//go:linkname AwsClientHook main.AwsClientHook
func AwsClientHook(client interface{}) {
	fmt.Printf("AWS Client Hook: Auto-instrumenting AWS client\n")
	fmt.Printf("Client instrumentation happens automatically when config is instrumented\n")
}
