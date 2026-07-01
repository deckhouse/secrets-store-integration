/*
Copyright 2026 Flant JSC

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

package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"sort"
)

// computeSecretsHash returns a deterministic digest of the resolved secrets.
// Both keys and values are hashed (keys from VAULT_ENV_FROM_PATH can change
// between polls, e.g. when a key is renamed in Vault), and every field is
// length-prefixed so that concatenation ambiguities can't produce collisions.
func computeSecretsHash(secrets map[string]string) string {
	if len(secrets) == 0 {
		return ""
	}

	// Keys are sorted to keep the hash deterministic despite random map iteration.
	keys := make([]string, 0, len(secrets))
	for key := range secrets {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	h := sha256.New()
	var lenBuf [8]byte
	writeField := func(s string) {
		binary.BigEndian.PutUint64(lenBuf[:], uint64(len(s)))
		h.Write(lenBuf[:])
		h.Write([]byte(s))
	}

	for _, key := range keys {
		writeField(key)
		writeField(secrets[key])
	}

	return hex.EncodeToString(h.Sum(nil))
}
