/*
Copyright 2024 Flant JSC

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

package webhook

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"

	"vault-secrets-webhook/pkg/common"
)

const (
	VaultEnvVolumeName = "stronghold-env"
)

func (mw *MutatingWebhook) MutatePod(ctx context.Context, pod *corev1.Pod, vaultConfig VaultConfig, dryRun bool) error {
	mw.logger.Debug("Successfully connected to the API")

	if isPodAlreadyMutated(pod) {
		mw.logger.Info(fmt.Sprintf("Pod %s is already mutated, skipping mutation.", pod.Name))
		return nil
	}

	initContainersMutated, err := mw.mutateContainers(ctx, pod.Spec.InitContainers, &pod.Spec, vaultConfig)
	if err != nil {
		return err
	}

	if initContainersMutated {
		mw.logger.Debug("Successfully mutated pod init containers")
	} else {
		mw.logger.Debug("No pod init containers were mutated")
	}

	containersMutated, err := mw.mutateContainers(ctx, pod.Spec.Containers, &pod.Spec, vaultConfig)
	if err != nil {
		return err
	}

	if containersMutated {
		mw.logger.Debug("Successfully mutated pod containers")
	} else {
		mw.logger.Debug("No pod containers were mutated")
	}

	containerEnvVars := []corev1.EnvVar{
		{
			Name:  "VAULT_ADDR",
			Value: vaultConfig.Addr,
		},
		{
			Name:  "VAULT_SKIP_VERIFY",
			Value: strconv.FormatBool(vaultConfig.SkipVerify),
		},
	}

	if vaultConfig.Token != "" {
		containerEnvVars = append(containerEnvVars, corev1.EnvVar{
			Name:  "VAULT_TOKEN",
			Value: vaultConfig.Token,
		})
	}

	containerVolMounts := []corev1.VolumeMount{
		{
			Name:      VaultEnvVolumeName,
			MountPath: "/vault/",
		},
	}
	if vaultConfig.TLSSecret != "" {
		mountPath := "/vault/tls/"
		volumeName := "vault-tls"
		if hasTLSVolume(pod.Spec.Volumes) {
			mountPath = "/stronghold-env/tls/"
			volumeName = "stronghold-env-tls"
		}

		containerEnvVars = append(containerEnvVars, corev1.EnvVar{
			Name:  "VAULT_CACERT",
			Value: mountPath + "ca.crt",
		})
		containerVolMounts = append(containerVolMounts, corev1.VolumeMount{
			Name:      volumeName,
			MountPath: mountPath,
		})
	}

	if initContainersMutated || containersMutated {

		pod.Spec.InitContainers = append(getInitContainers(pod.Spec.Containers, pod.Spec.SecurityContext, vaultConfig, initContainersMutated, containersMutated, containerEnvVars, containerVolMounts), pod.Spec.InitContainers...)
		mw.logger.Debug("Successfully appended pod init containers to spec")

		pod.Spec.Volumes = append(pod.Spec.Volumes, mw.getVolumes(pod.Spec.Volumes, vaultConfig)...)
		mw.logger.Debug("Successfully appended pod spec volumes")
	}

	return nil
}

func isPodAlreadyMutated(pod *corev1.Pod) bool {
	for _, volume := range pod.Spec.Volumes {
		if volume.Name == VaultEnvVolumeName {
			return true
		}
	}
	return false
}

func (mw *MutatingWebhook) mutateContainers(ctx context.Context, containers []corev1.Container, podSpec *corev1.PodSpec, vaultConfig VaultConfig) (bool, error) {
	mutated := false

	for i, container := range containers {
		var envVars []corev1.EnvVar
		if len(container.EnvFrom) > 0 {
			envFrom, err := mw.lookForEnvFrom(container.EnvFrom, vaultConfig.ObjectNamespace)
			if err != nil {
				return false, err
			}
			envVars = append(envVars, envFrom...)
		}

		for _, env := range container.Env {
			if common.HasVaultPrefix(env.Value) || HasInlineVaultDelimiters(env.Value) {
				envVars = append(envVars, env)
			}
			if env.ValueFrom != nil {
				valueFrom, err := mw.lookForValueFrom(env, vaultConfig.ObjectNamespace)
				if err != nil {
					return false, err
				}
				if valueFrom == nil {
					continue
				}
				envVars = append(envVars, *valueFrom)
			}
		}

		if len(envVars) == 0 && vaultConfig.VaultEnvFromPath == "" {
			continue
		}

		if func() bool {
			for _, skipContainerName := range vaultConfig.SkipMutateContainers {
				if container.Name == skipContainerName {
					return true
				}
			}
			return false
		}() {
			continue
		}

		mutated = true

		args := container.Command

		// the container has no explicitly specified command
		if len(args) == 0 {
			imageConfig, err := mw.registry.GetImageConfig(ctx, mw.k8sClient, vaultConfig.ObjectNamespace, vaultConfig.RegistrySkipVerify, &container, podSpec) //nolint:gosec
			if err != nil {
				return false, err
			}

			args = append(args, imageConfig.Entrypoint...)

			// If no Args are defined we can use the Docker CMD from the image
			// https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#notes
			if len(container.Args) == 0 {
				args = append(args, imageConfig.Cmd...)
			}
		}

		args = append(args, container.Args...)

		container.Command = []string{"/vault/env-injector"}
		container.Args = args

		// mutate probes if needed
		if vaultConfig.MutateProbes {
			// mutate LivenessProbe
			if container.LivenessProbe != nil && container.LivenessProbe.Exec != nil {
				lProbeCmd := container.LivenessProbe.Exec.Command
				container.LivenessProbe.Exec.Command = []string{"/vault/env-injector"}
				container.LivenessProbe.Exec.Command = append(container.LivenessProbe.Exec.Command, lProbeCmd...)
			}
			// mutate LivenessProbe
			if container.ReadinessProbe != nil && container.ReadinessProbe.Exec != nil {
				rProbeCmd := container.ReadinessProbe.Exec.Command
				container.ReadinessProbe.Exec.Command = []string{"/vault/env-injector"}
				container.ReadinessProbe.Exec.Command = append(container.ReadinessProbe.Exec.Command, rProbeCmd...)
			}
			// mutate StartupProbe
			if container.StartupProbe != nil && container.StartupProbe.Exec != nil {
				sProbeCmd := container.StartupProbe.Exec.Command
				container.StartupProbe.Exec.Command = []string{"/vault/env-injector"}
				container.StartupProbe.Exec.Command = append(container.StartupProbe.Exec.Command, sProbeCmd...)
			}
		}

		container.VolumeMounts = append(container.VolumeMounts, []corev1.VolumeMount{
			{
				Name:      VaultEnvVolumeName,
				MountPath: "/vault/",
			},
		}...)

		container.Env = append(container.Env, []corev1.EnvVar{
			{
				Name:  "VAULT_ADDR",
				Value: vaultConfig.Addr,
			},
			{
				Name:  "VAULT_SKIP_VERIFY",
				Value: strconv.FormatBool(vaultConfig.SkipVerify),
			},
			{
				Name:  "VAULT_AUTH_METHOD",
				Value: vaultConfig.AuthMethod,
			},
			{
				Name:  "VAULT_PATH",
				Value: vaultConfig.Path,
			},
			{
				Name:  "VAULT_ROLE",
				Value: vaultConfig.Role,
			},
			{
				Name:  "VAULT_IGNORE_MISSING_SECRETS",
				Value: vaultConfig.IgnoreMissingSecrets,
			},
			{
				Name:  "VAULT_ENV_PASSTHROUGH",
				Value: vaultConfig.VaultEnvPassThrough,
			},
			{
				Name:  "VAULT_JSON_LOG",
				Value: vaultConfig.EnableJSONLog,
			},
			{
				Name:  "VAULT_CLIENT_TIMEOUT",
				Value: vaultConfig.ClientTimeout.String(),
			},
		}...)

		if vaultConfig.Token != "" {
			container.Env = append(container.Env, corev1.EnvVar{
				Name:  "VAULT_TOKEN",
				Value: vaultConfig.Token,
			})
		}

		if !isLogLevelSet(container.Env) && vaultConfig.LogLevel != "" {
			container.Env = append(container.Env, []corev1.EnvVar{
				{
					Name:  "VAULT_LOG_LEVEL",
					Value: vaultConfig.LogLevel,
				},
			}...)
		}

		if len(vaultConfig.VaultNamespace) > 0 {
			container.Env = append(container.Env, []corev1.EnvVar{
				{
					Name:  "VAULT_NAMESPACE",
					Value: vaultConfig.VaultNamespace,
				},
			}...)
		}

		if vaultConfig.TLSSecret != "" {
			mountPath := "/vault/tls/"
			volumeName := "vault-tls"
			if hasTLSVolume(podSpec.Volumes) {
				mountPath = "/stronghold-env/tls/"
				volumeName = "stronghold-env-tls"
			}

			container.Env = append(container.Env, corev1.EnvVar{
				Name:  "VAULT_CACERT",
				Value: mountPath + "ca.crt",
			})
			container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
				Name:      volumeName,
				MountPath: mountPath,
			})
		} else if viper.GetString("cacert_bytes_b64") != "" {
			container.Env = append(container.Env, corev1.EnvVar{
				Name:  "VAULT_CACERT",
				Value: "/vault/default_ca.crt",
			})
		}

		if vaultConfig.VaultEnvFromPath != "" {
			container.Env = append(container.Env, corev1.EnvVar{
				Name:  "VAULT_ENV_FROM_PATH",
				Value: vaultConfig.VaultEnvFromPath,
			})
		}

		containers[i] = container
	}

	return mutated, nil
}

func (mw *MutatingWebhook) getVolumes(existingVolumes []corev1.Volume, vaultConfig VaultConfig) []corev1.Volume {
	mw.logger.Debug("Add generic volumes to podspec")

	volumes := []corev1.Volume{
		{
			Name: VaultEnvVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{
					Medium: corev1.StorageMediumMemory,
				},
			},
		},
	}

	if vaultConfig.TLSSecret != "" {
		mw.logger.Debug("Add vault TLS volume to podspec")

		volumeName := "vault-tls"
		if hasTLSVolume(existingVolumes) {
			volumeName = "stronghold-env-tls"
		}

		volumes = append(volumes, corev1.Volume{
			Name: volumeName,
			VolumeSource: corev1.VolumeSource{
				Projected: &corev1.ProjectedVolumeSource{
					Sources: []corev1.VolumeProjection{{
						Secret: &corev1.SecretProjection{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: vaultConfig.TLSSecret,
							},
							Items: []corev1.KeyToPath{{
								Key:  "ca.crt",
								Path: "ca.crt",
							}},
						},
					}},
				},
			},
		})
	}

	return volumes
}

// If the original Pod contained a Volume "vault-tls", for example Vault instances provisioned by the Operator
// we need to handle that edge case and choose another name for the vault-tls volume for accessing Vault with TLS.
func hasTLSVolume(volumes []corev1.Volume) bool {
	for _, volume := range volumes {
		if volume.Name == "vault-tls" {
			return true
		}
	}
	return false
}

func getInitContainers(originalContainers []corev1.Container, podSecurityContext *corev1.PodSecurityContext, vaultConfig VaultConfig, initContainersMutated bool, containersMutated bool, containerEnvVars []corev1.EnvVar, containerVolMounts []corev1.VolumeMount) []corev1.Container {
	containers := []corev1.Container{}

	if initContainersMutated || containersMutated {
		containers = append(containers, corev1.Container{
			Name:            "copy-env-injector",
			Image:           vaultConfig.EnvImage,
			ImagePullPolicy: vaultConfig.EnvImagePullPolicy,
			Command:         []string{"/bin/env-injector", "--self-copy"},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      VaultEnvVolumeName,
					MountPath: "/vault/",
				},
			},

			SecurityContext: getBaseSecurityContext(podSecurityContext, vaultConfig),
			Resources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    vaultConfig.EnvCPULimit,
					corev1.ResourceMemory: vaultConfig.EnvMemoryLimit,
				},
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    vaultConfig.EnvCPURequest,
					corev1.ResourceMemory: vaultConfig.EnvMemoryRequest,
				},
			},
		})

		if caB64 := viper.GetString("cacert_bytes_b64"); vaultConfig.TLSSecret == "" && caB64 != "" {
			containers[0].Env = append(containers[0].Env, corev1.EnvVar{
				Name:  "DEFAULT_CA_B64",
				Value: caB64,
			})
		}
	}

	return containers
}

func getBaseSecurityContext(podSecurityContext *corev1.PodSecurityContext, vaultConfig VaultConfig) *corev1.SecurityContext {
	context := &corev1.SecurityContext{
		AllowPrivilegeEscalation: &vaultConfig.PspAllowPrivilegeEscalation,
		ReadOnlyRootFilesystem:   &vaultConfig.ReadOnlyRootFilesystem,
		Capabilities: &corev1.Capabilities{
			Add: []corev1.Capability{},
			Drop: []corev1.Capability{
				"ALL",
			},
		},
	}

	if podSecurityContext != nil && podSecurityContext.RunAsUser != nil {
		context.RunAsUser = podSecurityContext.RunAsUser
	}

	// Although it could explicitly be set to false,
	// the behavior of false and unset are the same
	if vaultConfig.RunAsNonRoot {
		context.RunAsNonRoot = &vaultConfig.RunAsNonRoot
	}

	if vaultConfig.RunAsUser > 0 {
		context.RunAsUser = &vaultConfig.RunAsUser
	}

	if vaultConfig.RunAsGroup > 0 {
		context.RunAsGroup = &vaultConfig.RunAsGroup
	}

	return context
}

// isLogLevelSet checks if the VAULT_LOG_LEVEL environment variable
// has already been set in the container, so it doesn't get overridden.
func isLogLevelSet(envVars []corev1.EnvVar) bool {
	for _, envVar := range envVars {
		if envVar.Name == "VAULT_LOG_LEVEL" {
			return true
		}
	}
	return false
}
