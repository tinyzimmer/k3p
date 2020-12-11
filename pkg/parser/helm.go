package parser

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
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
	if vals := p.GetHelmValues(chart.Name()); vals != nil {
		raw, err := yaml.Marshal(p.HelmValues)
		if err != nil {
			return nil, err
		}
		helmVals, err = chartutil.ReadValues(raw)
		if err != nil {
			return nil, err
		}
		log.Debugf("Using the following values for chart %q: %+v\n", chart.Name(), helmVals)
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
	if vals := p.GetHelmValues(chart.Name()); vals != nil {
		log.Debugf("Marshaling helm values for chart %q: %+v\n", chart.Name(), vals)
		valuesBytes, err := yaml.Marshal(vals)
		if err != nil {
			return nil, err
		}
		valuesContent = string(valuesBytes)
	}

	// package the chart to a temp file
	var packagedChartBytes []byte
	var packagedChartName string
	if ok, err := chartutil.IsChartDir(chartPath); err == nil && ok {
		// Chart is a directory that needs to be packaged
		tmpDir, err := util.GetTempDir()
		if err != nil {
			return nil, err
		}
		defer os.RemoveAll(tmpDir)
		log.Debugf("Executing command: helm package %s -d %s\n", chartPath, tmpDir)
		_, err = exec.Command("helm", "package", chartPath, "-d", tmpDir).Output()
		if err != nil {
			return nil, err
		}
		files, err := ioutil.ReadDir(tmpDir)
		if err != nil {
			return nil, err
		}
		if len(files) != 1 {
			return nil, errors.New("helm package command produced more or less than 1 one artifact")
		}
		chartPkg := path.Join(tmpDir, files[0].Name())
		packagedChartName = path.Base(files[0].Name())
		packagedChartBytes, err = ioutil.ReadFile(chartPkg)
		if err != nil {
			return nil, err
		}
	} else {
		// Chart is already packaged
		log.Debugf("Chart at %q is already packaged, adding directly to manifest\n", chartPath)
		packagedChartName = path.Base(chartPath)
		packagedChartBytes, err = ioutil.ReadFile(chartPath)
		if err != nil {
			return nil, err
		}
	}

	stripExt := strings.TrimSuffix(path.Base(chartPath), ".tgz")

	var out bytes.Buffer
	if err := helmCRTmpl.Execute(&out, map[string]string{
		"Name":          strings.Replace(stripExt, ".", "-", -1),
		"Filename":      packagedChartName,
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
			Name: packagedChartName,
			Body: ioutil.NopCloser(bytes.NewReader(packagedChartBytes)),
			Size: int64(len(packagedChartBytes)),
		},
	}, nil
}
