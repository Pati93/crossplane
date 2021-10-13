/*
Copyright 2021 The Crossplane Authors.

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

package initializer

import (
	"context"

	"github.com/google/go-containerregistry/pkg/name"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/resource"

	v1 "github.com/crossplane/crossplane/apis/pkg/v1"
	"github.com/crossplane/crossplane/internal/xpkg"
)

const (
	errParsePackageName = "package name is not valid"
	errApplyPackage     = "cannot apply package"
)

// NewPackageInstaller returns a new package installer.
func NewPackageInstaller(p []string, c []string) *PackageInstaller {
	return &PackageInstaller{
		Providers:      p,
		Configurations: c,
	}
}

// PackageInstaller has the initializer for installing a list of packages.
type PackageInstaller struct {
	Configurations []string
	Providers      []string
}

// Run makes sure all specified packages exist.
func (pi *PackageInstaller) Run(ctx context.Context, kube client.Client) error {
	pkgs := make([]client.Object, len(pi.Providers)+len(pi.Configurations))
	// NOTE(hasheddan): we maintain a separate index from the range so that
	// Providers and Configurations can be added to the same slice for applying.
	pkgsIdx := 0
	for _, img := range pi.Providers {
		name, err := name.ParseReference(img, name.WithDefaultRegistry(""))
		if err != nil {
			return errors.Wrap(err, errParsePackageName)
		}
		p := &v1.Provider{
			ObjectMeta: metav1.ObjectMeta{
				Name: xpkg.ToDNSLabel(xpkg.ParsePackageSourceFromReference(name)),
			},
			Spec: v1.ProviderSpec{
				PackageSpec: v1.PackageSpec{
					Package: name.String(),
				},
			},
		}
		pkgs[pkgsIdx] = p
		pkgsIdx++
	}
	for _, img := range pi.Configurations {
		name, err := name.ParseReference(img, name.WithDefaultRegistry(""))
		if err != nil {
			return errors.Wrap(err, errParsePackageName)
		}
		c := &v1.Configuration{
			ObjectMeta: metav1.ObjectMeta{
				Name: xpkg.ToDNSLabel(xpkg.ParsePackageSourceFromReference(name)),
			},
			Spec: v1.ConfigurationSpec{
				PackageSpec: v1.PackageSpec{
					Package: name.String(),
				},
			},
		}
		pkgs[pkgsIdx] = c
		pkgsIdx++
	}
	pa := resource.NewAPIPatchingApplicator(kube)
	for _, p := range pkgs {
		if err := pa.Apply(ctx, p); err != nil {
			return errors.Wrap(err, errApplyPackage)
		}
	}
	return nil
}
