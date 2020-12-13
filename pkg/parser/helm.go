package parser

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
	"gopkg.in/yaml.v2"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"

	"github.com/tinyzimmer/k3p/pkg/log"
	"github.com/tinyzimmer/k3p/pkg/types"
	"github.com/tinyzimmer/k3p/pkg/util"
)

// TODO: Makes targetNamespace configurable for charts
var helmCRTmpl = template.Must(template.New("helm-cr").Funcs(sprig.TxtFuncMap()).Parse(`apiVersion: helm.cattle.io/v1
kind: HelmChart
metadata:
  name: {{ .Name }}
  namespace: kube-system
spec:
  targetNamespace: default 
  chart: https://%{KUBERNETES_API}%/static/k3p/{{ .Filename }}
{{- if .ValuesContent }}
  valuesContent: |-
    {{ .ValuesContent | nindent 4 }}
{{- end }}
`))

func isHelmArchive(file string) bool {
	log.Debug("Attempting to load", file, "as helm chart")
	_, err := loader.Load(file)
	return err == nil
}

func (p *ManifestParser) detectImagesFromHelmChart(chartPath string) ([]string, error) {
	images := make([]string, 0)

	chart, err := loader.Load(chartPath)
	if err != nil {
		return nil, err
	}

	var helmVals chartutil.Values
	if p.PackageConfig != nil {
		if vals, ok := p.PackageConfig.HelmValues[chart.Name()]; ok {
			raw, err := yaml.Marshal(vals)
			if err != nil {
				return nil, err
			}
			helmVals, err = chartutil.ReadValues(raw)
			if err != nil {
				return nil, err
			}
			log.Debugf("Using the following values for chart %q: %+v\n", chart.Name(), helmVals)
		}
	}

	if err := chartutil.ProcessDependencies(chart, helmVals); err != nil {
		return nil, err
	}

	options := chartutil.ReleaseOptions{
		Name:      "k3p-build",
		Namespace: "default",
		Revision:  1,
		IsInstall: true,
		IsUpgrade: false,
	}
	valuesToRender, err := chartutil.ToRenderValues(chart, helmVals, options, nil)
	if err != nil {
		return nil, err
	}
	if err != nil {
		return nil, err
	}

	log.Debugf("Rendering helm chart %q to kubernetes manifests\n", chart.Name())
	objects, err := engine.Render(chart, valuesToRender)
	if err != nil {
		return nil, err
	}

	// iterate all the yaml objects in the rendered templates
	for _, rendered := range objects {
		// Check if this is empty space
		if len(strings.TrimSpace(rendered)) == 0 {
			continue
		}
		// Decode the object
		obj, err := p.Decode([]byte(rendered))
		if err != nil {
			log.Debugf("Skipping invalid kubernetes object in rendered helm template: %s\n", err.Error())
			continue
		}
		// Append any images to the local images to be downloaded
		if objImgs := parseObjectForImages(obj); len(objImgs) > 0 {
			images = appendIfMissing(images, objImgs...)
		}
	}

	return images, nil
}

func (p *ManifestParser) packageHelmChartToArtifacts(chartPath string) ([]*types.Artifact, error) {
	chart, err := loader.Load(chartPath)
	if err != nil {
		return nil, err
	}

	var valuesContent string
	if p.PackageConfig != nil {
		rawValues, err := p.GetHelmValues(chart.Name())
		if err == nil {
			valuesContent = string(rawValues)
		} else {
			log.Debugf("Could not load helm values for chart %s: %s\n", chart.Name(), err.Error())
		}
	}

	// package the chart to a temp file
	var packagedChartBytes []byte
	var packagedChartFilename string
	if ok, err := chartutil.IsChartDir(chartPath); err == nil && ok {
		// Chart is a directory that needs to be packaged
		tmpDir, err := util.GetTempDir()
		if err != nil {
			return nil, err
		}
		defer os.RemoveAll(tmpDir)
		log.Debugf("Packaging helm chart %q to %q\n", chart.Name(), tmpDir)
		chartPkg, err := chartutil.Save(chart, tmpDir)
		if err != nil {
			return nil, err
		}
		log.Debugf("Produced chart package at %q\n", chartPkg)
		packagedChartFilename = path.Base(chartPkg)
		packagedChartBytes, err = ioutil.ReadFile(chartPkg)
		if err != nil {
			return nil, err
		}
	} else {
		// Chart is already packaged
		log.Debugf("Chart at %q is already packaged, adding directly to manifest\n", chartPath)
		packagedChartFilename = path.Base(chartPath)
		packagedChartBytes, err = ioutil.ReadFile(chartPath)
		if err != nil {
			return nil, err
		}
	}

	stripExt := strings.TrimSuffix(path.Base(chartPath), ".tgz")

	var out bytes.Buffer
	if err := helmCRTmpl.Execute(&out, map[string]string{
		"Name":          chart.Name(),
		"Filename":      packagedChartFilename,
		"ValuesContent": valuesContent,
	}); err != nil {
		return nil, err
	}
	outBytes := out.Bytes()
	return []*types.Artifact{
		{
			Type: types.ArtifactManifest,
			Name: fmt.Sprintf("%s-helm-chart.yaml", stripExt),
			Body: ioutil.NopCloser(bytes.NewReader(outBytes)),
			Size: int64(len(outBytes)),
		},
		{
			Type: types.ArtifactStatic,
			Name: packagedChartFilename,
			Body: ioutil.NopCloser(bytes.NewReader(packagedChartBytes)),
			Size: int64(len(packagedChartBytes)),
		},
	}, nil
}
