// Copyright (c) Flant JSC
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"fmt"
	"secrets-store-integration-hook/consts"

	tlscertificate "github.com/deckhouse/module-sdk/common-hooks/tls-certificate"
)

var _ = tlscertificate.RegisterInternalTLSHookEM(tlscertificate.GenSelfSignedTLSHookConf{
	CN:                    consts.WebhookName,
	TLSSecretName:         fmt.Sprintf("%s-tls", consts.WebhookName),
	Namespace:             consts.ModuleNamespace,
	CommonCACanonicalName: "Deckhouse",
	SANs: tlscertificate.DefaultSANs([]string{
		fmt.Sprintf("%s.%s.svc", consts.WebhookName, consts.ModuleNamespace),
	}),
	FullValuesPathPrefix: fmt.Sprintf("%s.internal.webhookCert", consts.DotValuesModuleName),
})
