---
title: "Module secrets-store-integration"
description: "Overview of the secrets-store-integration module for delivering secrets from Vault-compatible stores to Kubernetes workloads."
---

The `secrets-store-integration` module is intended for teams that store application secrets in
Deckhouse Stronghold or another Vault-compatible secret store and need to deliver them to workloads
running in Kubernetes.

The module connects workloads to an external secret store and helps avoid manual secret
distribution across clusters. It supports delivering secrets as files and environment variables.

## Main Features

- Delivers secrets from Vault-compatible stores into workloads running in Kubernetes.
- Mounts secrets, keys, and certificates into containers as files using the CSI driver.
- Injects secrets into application processes as environment variables at startup.
- Supports automatic connection to local `stronghold` and manual connection to an external store.
- Uses Kubernetes `ServiceAccount` authentication for workload access to the secret store.
- Rotates mounted secrets automatically when the source value changes.
- Supports delivery of Base64-encoded binary files through `SecretsStoreImport`.

## Limitations

- The module supports only secret stores compatible with the HashiCorp Vault API.
- Mounted secret rotation does not work for files mounted with `subPath`.
