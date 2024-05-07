---
title: "The secrets-store-integration module"
description: "The secrets-store-integration module provides integration of secrets stores and applications in the k8s clusters"
---

The secrets-store-integration module implements the delivery of secrets to application pods in the Kubernetes
cluster by mounting multiple secrets, keys, and certificates stored in external secrets stores.

Secrets are mounted into pods as a volume using CSI driver implementation.
Secrets stores must be compatible with Hashicorp Vault API.

## Delivering secrets to the applications

There are several ways to deliver secrets to an application from vault-compatible storage:

1. Your application accesses the vault itself.
*Recommendation:* This is the most secure option, but requires application modification.

2. A layered application accesses the vault, and your application accesses secrets from files created in the container.
*Recommendation:* If there is no possibility to modify applications, use this option. It is less secure because the secret data is stored in files in the container, but it is easier to implement.

3. The storage is accessed by the layering application, and your application gets access to the secrets from the environment variables.
*Recommendation:* If you can't read from files, you can use this option, but it is NOT secure, because the secret data is stored in Kubernetes (and etcd, so it can potentially be read on any node in the cluster).

<thead>
<tr>
<th>How secrets are being delivered?</th>
<th>Resources consumption</th>
<th>How your application gets the data?</th>
<th>Where the secret is stored in the Kubernetes?</th>
<th>Статус</th>
</tr>
</thead>
<tbody>
<tr>
<td><a style="color: ##0066FF;" href="#option-1-get-the-secrets-from-the-app-itself">App</a></td>
<td>Doesn't change</td>
<td>Directly from secrets store</td>
<td>Is not stored</td>
<td>Implemented</td>
</tr>
<tr>
<td><a style="color: ##0066FF;" href="#CSI-interface">CSI Interface</a></td>
<td>Two pods per node (daemonset)</td>
<td><ul><li>From disk volume (as a file)</li><li>From environment variable</li></ul></td>
<td>Is not stored</td>
<td>Implemented</td>
</tr>
<tr>
<td><a style="color: ##0066FF;" href="#option-3-entrypoint-injection">Entrypoint injection</a></td>
<td>One app for whole cluster (deployment)</td>
<td>Secrets are delivered as environment variables during application start</td>
<td>Is not stored</td>
<td>In the process of implementation</td>
</tr>
<tr>
<td><a style="color: ##0066FF;" href="#option-4-delivering-secrets-through-Kubernetes-mechanisms">Kubernetes Secrets</a></td>
<td>One app for whole cluster (deployment)</td>
<td><ul><li>From disk volume (as a file)</li><li>From environment variable</li></ul></td>
<td>Stored as a Kubernetes Secret</td>
<td>Planned for implementation and release</td>
</tr>
<tr>
<td><a style="color: #A9A9A9; font-style: italic;" href="#for-reference-vault-agent-injector">Vault-agent Injector</a></td>
<td style="color: #A9A9A9; font-style: italic;">One agent per one pod (sidecar)</td>
<td style="color: #A9A9A9; font-style: italic;">From disk volume (as a file)</td>
<td style="color: #A9A9A9; font-style: italic;">Is not stored</td>
<td style="color: #A9A9A9; font-style: italic;"><sup><b>*</b></sup>No plans to implement</td>
</tr>
</tbody>
</table>

<i><sup>*</sup>No plans to implement. There are no advantages over the CSI interface.</i>

### Option #1: Get the secrets from the app itself

> *Status:* The most secure option. Recommended for use if there is a possibility of application modification.

The application accesses Stronghold API and requests the necessary secret via HTTPS protocol using authorization token (token from SA).

#### Pros:

- The secret received by the application is not stored anywhere except in the application itself, there is no danger that it will be compromised during transmission

#### Cons:

- Requires customization of the application to be able to work with Stronghold
- Requires reimplementation of secret access in each application, and if library is updated, rebuilds all applications
- Application must support TLS and certificate validation
- No caching, when restarting the application you need to re-request the secret directly from the repository

### Option #2: Deliver secrets through files

#### CSI interface

> *Status:* Secure option. Recommended for use if there is no way to modify applications.

When creating pods requesting CSI volumes, the CSI secret vault driver sends a request to Vault CSI. Vault CSI then uses the specified SecretProviderClass and ServiceAccount feed to retrieve secrets from the vault and mount them in the pod volume.

#### Environment Variable Injection:

In a situation where there is no way to modify the application code, then you can implement secure secret injection as an environment variable for the application. To do this, read all the files CSI has mounted in the container and define environment variables with names corresponding to the file names and values corresponding to the contents of the files. After that, run the original application.

Example in bash:

```bash
bash -c "for file in $(ls /mnt/secrets); do export  $file=$(cat /mnt/secrets/$file); done ; exec my_original_file_to_startup"
```

#### Pros:

- Only two containers with predictable resources on each node to serve the secret delivery system to applications.
- Creating SecretsStore/SecretProviderClass resources reduces the amount of repetitive code compared to other vault agent implementations.
- It is possible to create a copy of the secret from the vault as a kubernetes secret if needed.
- The secret is retrieved from the vault by the CSI driver during the container creation phase. This means that starting pods will be blocked until the secrets are read from the repository and written to the volume of the container.

### Option №3: Entrypoint injection

#### Environment variables delivery into the container through entrypoint injection

> *Статус:* Secure option. In the process of implementation

Environment variables are being delivered into the container during the application start and are located only in memory. During the first stage of implementation delivery will be made with the entrypoint injection. Afterwards, delivery mechanism will be integrated into containerd.

### Option #4: Delivering secrets through Kubernetes mechanisms

> *Status:* Not a safe option, not recommended for use. No support is available, but it is planned for the future.

This integration method implements a Kubernetes secrets operator with a set of CRDs responsible for synchronizing secrets from Vault to Kubernetes secrets.

#### Minuses:

- The secret is located both in the secret store and the Kubernetes secret. It is accessible through the Kubernetes API, as well as in Etcd, which means it can potentially be read on any node in the cluster or extracted from an Etcd backup. The secret data will be always stored as Kubernetes secret.

#### Pros:

- The classic way to transfer a secret to an application through environment variables is to connect the Kubernetes secret.

### For reference: vault-agent injector

> *Status:* Has no pros compared to the CSI mechanism. No support and implementation is available or planned.

When a pod is created, a mutation occurs that adds a vault-agent container. The vault-agent accesses the secret store, retrieves the secret, and places it into a shared volume on a disk that can be accessed by the application.

#### Minuses:

- Each pod needs a sidecar, which consumes resources in one way or another. Let's imagine a cluster with 50 applications, each with 3 to 15 replicas. The sidecar with an agent needs to allocate CPU and memory resources. Although small, it is very noticeable. 50mcpu + 100Mi for one sidecar and in total for all applications will consume tens of cores and tens of gigabytes of RAM.
- Since we are monitoring metrics from each container, with this approach we will get x2 metrics just by container.
