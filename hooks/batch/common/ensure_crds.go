// Copyright (c) Flant JSC
// SPDX-License-Identifier: Apache-2.0

package common

import (
	ensure_crds "github.com/deckhouse/module-sdk/common-hooks/ensure_crds"
)

var _ = ensure_crds.RegisterEnsureCRDsHookEM("../crds/*.yaml")
var _ = ensure_crds.RegisterEnsureCRDsHookEM("../crds/*/*.yaml")
