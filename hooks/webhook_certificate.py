#!/usr/bin/env python3
#
# Copyright 2024 Flant JSC
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
from cryptography.hazmat.primitives import serialization
from cryptography.hazmat.primitives.asymmetric import rsa
from cryptography.hazmat.backends import default_backend
from cryptography import x509
from cryptography.x509.oid import NameOID
from cryptography.hazmat.primitives import serialization, hashes
from datetime import datetime, timedelta

config = """
configVersion: v1
onStartup: 10
beforeHelm: 10
"""

def main(ctx: hook.Context):

    if not ctx.values["secretsStoreIntegration"]["internal"].get("webhookCert"):

        private_key_ca = rsa.generate_private_key(
            public_exponent=65537,
            key_size=2048,
            backend=default_backend()
        )

        subject_ca = issuer_ca = x509.Name([
            x509.NameAttribute(NameOID.COMMON_NAME, "secrets-store-integration"),
        ])

        ca = x509.CertificateBuilder().subject_name(
            subject_ca
        ).issuer_name(
            subject_ca
        ).public_key(
            private_key_ca.public_key()
        ).serial_number(
            x509.random_serial_number()
        ).not_valid_before(
            datetime.utcnow()
        ).not_valid_after(
            datetime.utcnow() + timedelta(days=3650)
        ).add_extension(
            x509.BasicConstraints(ca=True, path_length=None),
            critical=True,
        ).sign(private_key_ca, hashes.SHA256(), default_backend())

        private_key_csr = rsa.generate_private_key(
            public_exponent=65537,
            key_size=2048,
            backend=default_backend()
        )

        subject_csr = x509.Name([
            x509.NameAttribute(NameOID.COMMON_NAME, "vault-secrets-webhook"),
        ])

        csr = x509.CertificateSigningRequestBuilder().subject_name(
            subject_csr
        ).sign(private_key_csr, hashes.SHA256(), default_backend())

        signed_cert = x509.CertificateBuilder().subject_name(
            subject_csr
        ).issuer_name(
            ca.issuer
        ).public_key(
            csr.public_key()
        ).serial_number(
            x509.random_serial_number()
        ).not_valid_before(
            datetime.utcnow()
        ).not_valid_after(
            datetime.utcnow() + timedelta(days=3650)
        ).add_extension(
            x509.BasicConstraints(ca=False, path_length=None),
            critical=True,
        ).add_extension(
        x509.SubjectAlternativeName([
            x509.DNSName("vault-secrets-webhook.d8-secrets-store-integration.svc"),
        ]),
        critical=False,
        ).sign(private_key_ca, hashes.SHA256(), default_backend())

        ca_pub = (ca.public_bytes(
            serialization.Encoding.PEM)).decode('utf-8')
        client_pub = (signed_cert.public_bytes(
            serialization.Encoding.PEM)).decode('utf-8')
        client_key = (private_key_csr.private_bytes(
            encoding=serialization.Encoding.PEM,
            format=serialization.PrivateFormat.PKCS8,
            encryption_algorithm=serialization.NoEncryption())).decode('utf-8')

        ctx.values["secretsStoreIntegration"]["internal"]["webhookCert"]["ca"] = ca_pub
        ctx.values["secretsStoreIntegration"]["internal"]["webhookCert"]["crt"] = client_pub
        ctx.values["secretsStoreIntegration"]["internal"]["webhookCert"]["key"] = client_key


if __name__ == "__main__":
    hook.run(main, config=config)
