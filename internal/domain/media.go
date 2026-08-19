package domain

import "strings"

const (
	MediaOCIManifest    = "application/vnd.oci.image.manifest.v1+json"
	MediaOCIIndex       = "application/vnd.oci.image.index.v1+json"
	MediaOCIConfig      = "application/vnd.oci.image.config.v1+json"
	MediaOCILayer       = "application/vnd.oci.image.layer.v1.tar"
	MediaOCIArtifact    = "application/vnd.oci.artifact.manifest.v1+json"
	MediaDockerManifest = "application/vnd.docker.distribution.manifest.v2+json"
)

func IsManifestMediaType(v string) bool {
	switch strings.Split(v, ";")[0] {
	case MediaOCIManifest, MediaOCIIndex, MediaOCIArtifact, MediaDockerManifest:
		return true
	}
	return false
}
func IsBlobMediaType(v string) bool { return strings.TrimSpace(v) != "" }
func ValidateManifestMediaType(v string) error {
	if !IsManifestMediaType(v) {
		return ErrInvalidMediaType
	}
	return nil
}
func NormalizeMediaType(v string) string { return strings.TrimSpace(strings.Split(v, ";")[0]) }
