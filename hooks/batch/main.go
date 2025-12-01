// Copyright (c) Flant JSC
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"github.com/deckhouse/module-sdk/pkg/app"

	_ "secrets-store-integration-hook/certificates"
	_ "secrets-store-integration-hook/common"
	_ "secrets-store-integration-hook/ssi-spc-label-migrate"
)

func main() {
	app.Run()
}
