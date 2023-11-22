// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package internal

// This file is a script which generates the .goreleaser.yaml file for all
// supported OpenTelemetry Collector distributions.
//
// Run it with `make generate-goreleaser`.

import (
	"fmt"
	"path"
	"strings"

	"github.com/goreleaser/goreleaser/pkg/config"
)

var (
	ImagePrefixes = []string{"otel"}
	Architectures = []string{"amd64", "arm64"}
	ArmVersions   = []string{"7"}
)

func Generate(imagePrefixes []string, dists []string) config.Project {
	return config.Project{
		ProjectName: "opentelemetry-collector-releases",
		Checksum: config.Checksum{
			NameTemplate: "{{ .ProjectName }}_checksums.txt",
		},

		Builds:          Builds(dists),
		Archives:        Archives(dists),
		Dockers:         DockerImages(imagePrefixes, dists),
		DockerManifests: DockerManifests(imagePrefixes, dists),
	}
}

func Builds(dists []string) (r []config.Build) {
	for _, dist := range dists {
		r = append(r, Build(dist))
	}
	return
}

// Build configures a goreleaser build.
// https://goreleaser.com/customization/build/
func Build(dist string) config.Build {
	return config.Build{
		ID:     dist,
		Dir:    path.Join("distributions", dist, "_build"),
		Binary: dist,
		BuildDetails: config.BuildDetails{
			Env:     []string{"CGO_ENABLED=0"},
			Flags:   []string{"-trimpath"},
			Ldflags: []string{"-s", "-w"},
		},
		Goos:   []string{"darwin", "linux"},
		Goarch: Architectures,
		Goarm:  ArmVersions,
	}
}

func Archives(dists []string) (r []config.Archive) {
	for _, dist := range dists {
		r = append(r, Archive(dist))
	}
	return
}

// Archive configures a goreleaser archive (tarball).
// https://goreleaser.com/customization/archive/
func Archive(dist string) config.Archive {
	return config.Archive{
		ID:           dist,
		NameTemplate: "{{ .Binary }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}{{ if .Mips }}_{{ .Mips }}{{ end }}",
		Builds:       []string{dist},
	}
}

func DockerImages(imagePrefixes, dists []string) (r []config.Docker) {
	for _, dist := range dists {
		// Only support amd64 for Docker images.
		r = append(r, DockerImage(imagePrefixes, dist, "amd64", ""))
	}
	return
}

// DockerImage configures goreleaser to build a container image.
// https://goreleaser.com/customization/docker/
func DockerImage(imagePrefixes []string, dist, arch, armVersion string) config.Docker {
	dockerArchName := arch
	var imageTemplates []string
	for _, prefix := range imagePrefixes {
		dockerArchTag := strings.ReplaceAll(dockerArchName, "/", "")
		imageTemplates = append(
			imageTemplates,
			fmt.Sprintf("%s/%s:{{ .Version }}-%s", prefix, imageName(dist), dockerArchTag),
			fmt.Sprintf("%s/%s:latest-%s", prefix, imageName(dist), dockerArchTag),
		)
	}

	label := func(name, template string) string {
		return fmt.Sprintf("--label=org.opencontainers.image.%s={{%s}}", name, template)
	}

	return config.Docker{
		ImageTemplates: imageTemplates,
		Dockerfile:     path.Join("distributions", dist, "Dockerfile"),

		Use: "buildx",
		BuildFlagTemplates: []string{
			"--pull",
			fmt.Sprintf("--platform=linux/%s", dockerArchName),
			label("created", ".Date"),
			label("name", ".ProjectName"),
			label("revision", ".FullCommit"),
			label("version", ".Version"),
			label("source", ".GitURL"),
		},
		Files:  []string{path.Join("configs", fmt.Sprintf("%s.yaml", dist))},
		Goos:   "linux",
		Goarch: arch,
		Goarm:  armVersion,
	}
}

func DockerManifests(imagePrefixes, dists []string) (r []config.DockerManifest) {
	for _, dist := range dists {
		for _, prefix := range imagePrefixes {
			r = append(r, DockerManifest(prefix, `{{ .Version }}`, dist))
			r = append(r, DockerManifest(prefix, "latest", dist))
		}
	}
	return
}

// DockerManifest configures goreleaser to build a multi-arch container image manifest.
// https://goreleaser.com/customization/docker_manifest/
func DockerManifest(prefix, version, dist string) config.DockerManifest {
	return config.DockerManifest{
		NameTemplate: fmt.Sprintf("%s/%s:%s", prefix, imageName(dist), version),
		ImageTemplates: []string{
			// Only support amd64 for Docker images.
			fmt.Sprintf("%s/%s:%s-%s", prefix, imageName(dist), version, "amd64"),
		},
	}
}

// imageName translates a distribution name to a container image name.
func imageName(dist string) string {
	return strings.Replace(dist, "otelcol", "opentelemetry-collector", 1)
}
