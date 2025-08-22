package api

type SecretStoreImportSpec struct {
	AuthPath      string `json:"authPath"`
	Namespace     string `json:"namespace"`
	Address       string `json:"address"`
	CACert        string `json:"caCert"`
	Audience      string `json:"audience"`
	SkipTLSVerify bool   `json:"skipTLSVerify"`
	Files         []struct {
		Name   string `json:"name"`
		Source struct {
			Key  string `json:"key"`
			Path string `json:"path"`
		} `json:"source"`
	} `json:"files"`
	Role string `json:"role"`
	Type string `json:"type"`
}

type SecretStoreImportMetadata struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type SecretStoreImport struct {
	Metadata SecretStoreImportMetadata `json:"metadata"`
	Spec     SecretStoreImportSpec     `json:"spec"`
}
