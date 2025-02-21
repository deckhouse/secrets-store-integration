#!/usr/bin/env python3
#
# Copyright 2023 Flant JSC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


from deckhouse import hook
import yaml
import json

config = """
configVersion: v1
kubernetes:
- name: secrets-store
  apiVersion: "deckhouse.io/v1alpha1"
  kind: "SecretsStoreImport"
  queue: "/modules/secrets-store-integration/secrets-store"
  keepFullObjectsInMemory: false
  jqFilter: |
    { metadata: .metadata, spec:.spec }
"""

template = """
apiVersion: secrets-store.csi.x-k8s.io/v1
kind: SecretProviderClass
metadata:
  name: null
  namespace: default
  labels:
    heritage: deckhouse
    module: secrets-store-integration
spec:
  provider: vault
  parameters:
    roleName: null
    objects: ""
  secretObjects:
  - secretName: ""
    type: Opaque
    data: []
"""

def main(ctx: hook.Context):
    if ctx.binding_context['type'] == 'Synchronization':
        for ssi in ctx.binding_context['objects']:
            name = ssi['filterResult']['metadata']['name']
            namespace = ssi['filterResult']['metadata']['namespace']
            sps = yaml.safe_load(template)
            sps['metadata']['name'] = name
            sps['metadata']['namespace'] = namespace
            sps['spec']['parameters']['roleName'] = ssi['filterResult']['spec']['role']
            sps['spec']['secretObjects'][0]['secretName'] = name
            for obj in ssi['filterResult']['spec']['files']:
                sps['spec']['parameters']['objects'] += '- objectName: ' + \
                    obj['name'] + "\n"
                sps['spec']['parameters']['objects'] += '  secretPath: ' + \
                    obj['source']['path'] + "\n"
                sps['spec']['parameters']['objects'] += '  secretKey: ' + \
                    obj['source']['key'] + "\n"
                sps['spec']['secretObjects'][0]['data'].append(
                    {"key": obj['source']['key'], "objectName": obj['name']})
            ctx.kubernetes.create_or_update(sps)

    if ctx.binding_context['type'] == 'Event':
        event = ctx.binding_context['watchEvent']
        name = ctx.binding_context['filterResult']['metadata']['name']
        namespace = ctx.binding_context['filterResult']['metadata']['namespace']

        if event == 'Added' or event == 'Modified':
            sps = yaml.safe_load(template)
            sps['metadata']['name'] = name
            sps['metadata']['namespace'] = namespace
            sps['spec']['parameters']['roleName'] = ctx.binding_context['filterResult']['spec']['role']
            sps['spec']['secretObjects'][0]['secretName'] = name
            for obj in ctx.binding_context['filterResult']['spec']['files']:
                sps['spec']['parameters']['objects'] += '- objectName: ' + \
                    obj['name'] + "\n"
                sps['spec']['parameters']['objects'] += '  secretPath: ' + \
                    obj['source']['path'] + "\n"
                sps['spec']['parameters']['objects'] += '  secretKey: ' + \
                    obj['source']['key'] + "\n"
                sps['spec']['secretObjects'][0]['data'].append(
                    {"key": obj['source']['key'], "objectName": obj['name']})
            ctx.kubernetes.create_or_update(sps)
        elif event == 'Deleted':
            ctx.kubernetes.delete('SecretProviderClass', namespace, name)


if __name__ == "__main__":
    hook.run(main, config=config)
