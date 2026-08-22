package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/kaizenforyou91/forge/pkg/app"
	"github.com/kaizenforyou91/forge/pkg/compiler"
	"github.com/kaizenforyou91/forge/pkg/manifest"
	"github.com/kaizenforyou91/forge/pkg/registry"
	"github.com/spf13/cobra"
)

func newBuildCmd() *cobra.Command {
	var outputPath string

	cmd := &cobra.Command{
		Use:   "build <manifest>",
		Short: "Compile and package a Forge manifest",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			application, err := ApplicationFromContext(cmd.Context())
			if err != nil {
				return err
			}

			manifestPath := args[0]

			m, err := loadManifestFile(manifestPath)
			if err != nil {
				return err
			}

			registryInstance, err := resolveRegistry(application)
			if err != nil {
				return err
			}

			sourceRegistry, err := resolveSourceRegistry(application)
			if err != nil {
				return err
			}

			engine, err := resolveEngine(application)
			if err != nil {
				return err
			}

			if strings.TrimSpace(outputPath) == "" {
				outputPath = filepath.Join(
					"build",
					m.Name+"-"+m.Version+".zip",
				)
			}

			admission, err := compiler.AdmitManifest(
				m,
				registryInstance,
				sourceRegistry,
			)
			if err != nil {
				return err
			}

			if err := compiler.CompileAndPackagePlan(
				engine,
				admission.BuildPlan(),
				compiler.NewZIPPackager(),
				outputPath,
			); err != nil {
				return err
			}

			fmt.Fprintf(
				cmd.OutOrStdout(),
				"Build completed: %s\n",
				outputPath,
			)

			return nil
		},
	}

	cmd.Flags().StringVar(
		&outputPath,
		"output",
		"",
		"Output package path",
	)

	return cmd
}

func loadManifestFile(path string) (manifest.Manifest, error) {
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".yaml", ".yml":
		return manifest.LoadYAML(path)

	case ".json":
		return manifest.LoadJSON(path)

	default:
		return manifest.Manifest{}, fmt.Errorf(
			"unsupported manifest format %q",
			ext,
		)
	}
}

func resolveRegistry(
	application *app.App,
) (*registry.Registry, error) {
	if application == nil {
		return nil, fmt.Errorf("application is nil")
	}

	var value *registry.Registry

	if err := application.Container().Resolve(&value); err != nil {
		return nil, err
	}

	if value == nil {
		return nil, fmt.Errorf("compiler registry is nil")
	}

	return value, nil
}

func resolveEngine(
	application *app.App,
) (*compiler.Engine, error) {
	if application == nil {
		return nil, fmt.Errorf("application is nil")
	}

	var value *compiler.Engine

	if err := application.Container().Resolve(&value); err != nil {
		return nil, err
	}

	if value == nil {
		return nil, fmt.Errorf("compiler engine is nil")
	}

	return value, nil
}

func resolveSourceRegistry(
	application *app.App,
) (*compiler.PackageSourceRegistry, error) {
	if application == nil {
		return nil, fmt.Errorf("application is nil")
	}

	var value *compiler.PackageSourceRegistry

	if err := application.Container().Resolve(&value); err != nil {
		return nil, err
	}

	if value == nil {
		return nil, fmt.Errorf("compiler package source registry is nil")
	}

	return value, nil
}
