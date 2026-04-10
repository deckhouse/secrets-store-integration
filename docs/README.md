---
title: "The secrets-store-integration module"
description: "The secrets-store-integration module integrates secret stores and applications in K8s clusters"
---

The `secrets-store-integration` module delivers secrets to applications in Kubernetes clusters.

It allows applications to receive secrets from an external secret store compatible with the HashiCorp Vault API.

## Main Features

- Delivers secrets to pods as mounted files or environment variables without storing them in Kubernetes.
- Supports automatic connection to the internal Deckhouse Stronghold instance with zero manual configuration (`DiscoverLocalStronghold` mode).
- Works with any Vault-compatible secret store in `Manual` mode.
- Automatically rotates mounted secrets every two minutes when the value in the store changes.
- Provides entrypoint injection for applications that cannot be modified to read secrets directly from the store.
- Supports Base64-encoded binary secrets (e.g. JKS keystores, Kerberos keytab files) with automatic decoding on delivery.

Secrets can be delivered to an application in different ways. To avoid mixing different concepts, this documentation separates them into three levels:

- **architectural model**: who retrieves the secret from the external store;
- **delivery form**: how the application reads the secret;
- **implementation mechanism**: how this is implemented in Kubernetes.

This distinction matters because an application, CSI, entrypoint injection, and environment variables are not entities of the same level.

## Architectural models

### Application retrieves the secret itself

In this scenario, the application addresses the external store directly and gets the secret without any intermediate storage in Kubernetes.

This is the most secure option. It is the recommended approach if the application can be modified.

### Platform delivers the secret to the application through a file

In this scenario, an infrastructure component retrieves the secret, and the application reads it from a file mounted into the container.

The main implementation mechanism is CSI. In the [comparison table](#scenario-comparison), this is the scenario where the application reads data from a disk volume and the secret is not stored in Kubernetes.

### The platform delivers the secret to the application through environment variables

In this scenario, an infrastructure component retrieves the secret, and the application sees it as an environment variable.

One implemented approach is entrypoint injection: secrets are delivered from the store at application startup as environment variables, and they are not stored in Kubernetes.

## Secret delivery forms

### Via a file

The application reads the secret from a file in a mounted volume.

This scenario is implemented through CSI. The CSI driver retrieves the secret from the store during container creation, so pod startup is blocked until the secrets are read from the store and written to the volume.

### Via environment variables

The application reads the secret from environment variables.

This uses the injector: if a pod has the `secrets-store.deckhouse.io/role` annotation, the mutating webhook modifies the pod manifest, adds an init container, and replaces the container startup command with the injector. The injector retrieves secrets from a Vault-compatible store, puts them into the process `ENV`, and then starts the original command through `execve`.

If a container does not define a startup command in its manifest, the command is taken from the image manifest in the registry.

## Implementation mechanisms

### CSI

CSI is the primary mechanism for delivering secrets as files.

For the CSI scenario:

- the application reads the secret from a disk volume as a file;
- the secret is not stored in Kubernetes;
- pod startup depends on successfully reading the secret and writing it to the volume.

### Entrypoint injection

Entrypoint injection is a mechanism for delivering secrets into environment variables at application startup.

From the application's point of view, this is a separate scenario of consuming a secret through `ENV`, not through a file.

### Environment variable injector

The environment variable injector is the technical mechanism that implements delivery into `ENV` through a mutating webhook, an init container, and launching the original command with `execve`.

If both of the following are used at the same time:

- the `secrets-store.deckhouse.io/env-from-path` annotation;
- and an explicitly defined environment variable with the same name,

the value from the `env-from-path` annotation takes precedence.

## Scenario comparison

| Architectural model | Implementation mechanism | How the application gets the data | Where it is stored in Kubernetes | Resource consumption |
| --- | --- | --- | --- | --- |
| The application retrieves the secret itself | Direct application access | Directly from the secret store | Not stored | Unchanged |
| The platform delivers the secret through a file | CSI | From a disk volume (as a file) | Not stored | Two pods on each node (DaemonSet) |
| The platform delivers the secret through environment variables | Entrypoint injection | Secrets are delivered from the store at application startup as environment variables | Not stored | One pod on each node (DaemonSet) |

## What to consider when choosing

- If the application can be modified, direct application access to the store is preferred as the most secure option.
- If the application can read secrets from files, use delivery through CSI.
- If the application cannot be changed and requires environment variables, use environment variable injection.

## Limitations and specifics

- When CSI is used, pod startup is blocked until secrets are read from the store and written to the volume.
- In approaches that use additional containers, the number of container metrics increases because metrics are collected from every container.
