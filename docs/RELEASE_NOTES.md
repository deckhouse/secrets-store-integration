---
title: "Release Notes"
---

## v1.4.1

* Updated base images to v1.3.5
* Updated changelog generator

## v1.4.0

* Added secret refresh mode and lease refresh mode for env-injector
* Added ability to set service-account-token-path
* Changed slog to deckhouse logger
* Added secret templating
* Removed bank-vaults dependencies
* Removed custom transit engine secret decode
* Updated documentation, added usage examples

## v1.3.18

* Upgraded base images from v0.5.77 to v1.1.11 and adjusted image build definitions to use updated builder/import paths.
* Added VEX attestation infrastructure for module builds
* CVE-2026-25680, CVE-2026-25681, CVE-2026-27136, CVE-2026-27145, CVE-2026-33814, CVE-2026-39821, CVE-2026-39826, CVE-2026-39827, CVE-2026-39828, CVE-2026-39829, CVE-2026-39830, CVE-2026-39835, CVE-2026-39883, CVE-2026-42502, CVE-2026-42504, CVE-2026-42506, CVE-2026-42508, CVE-2026-46595, CVE-2026-46597, CVE-2026-39831, CVE-2026-39832, CVE-2026-39833, CVE-2026-39834, CVE-2026-42507, CVE-2026-46598
* {'Added security-related CI coverage': 'antivirus scan, Gitleaks PR/daily scans, and scheduled Svace analysis'}
* {'Updated module metadata': 'Deckhouse requirement is now >= 1.72 and the module is registered under the security subsystem'}
* Documentation updates

## v1.3.17

* Updated lib_helm to 1.71.11
* Updated base images to v0.5.77
* CVE-2026-39883, CVE-2026-29181
* DOCS - refactoring secret-store-integration start pages and usage pages
* CI - Added changelog generator
