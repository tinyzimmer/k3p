package parser

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"

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
  chartContent: {{ .ChartContent }}
{{- if .ValuesContent }}
  valuesContent: |-
    {{ .ValuesContent | nindent 4 }}
{{- end }}
`))

func isHelmChart(dir string) bool {
	info, err := os.Stat(path.Join(dir, "Chart.yaml"))
	return err == nil && !info.IsDir()
}

func (p *ManifestParser) detectImagesFromHelmChart(chartPath string) ([]string, error) {
	images := make([]string, 0)

	args := []string{"template", chartPath}
	if helmArgs := p.GetHelmArgs(); helmArgs != "" {
		args = append(args, strings.Fields(helmArgs)...)
	}

	log.Debugf("Executing command: helm %s", strings.Join(args, " "))
	out, err := exec.Command("helm", args...).Output()
	if err != nil {
		return nil, err
	}

	// iterate all the yaml objects in the file
	rawYamls := strings.Split(string(out), "---")
	for _, raw := range rawYamls {
		// Check if this is empty space
		if strings.TrimSpace(raw) == "" {
			continue
		}
		// Decode the object
		obj, err := p.Decode([]byte(raw))
		if err != nil {
			log.Debugf("Skipping invalid kubernetes object in rendered helm template: %s", err.Error())
			continue
		}
		// Append any images to the local images to be downloaded
		if objImgs := parseObjectForImages(obj); len(objImgs) > 0 {
			images = appendIfMissing(images, objImgs...)
		}
	}

	return images, nil
}

func (p *ManifestParser) packageHelmChartToManifest(chartPath string) (*types.Artifact, error) {

	// only support values files for now, need to find a better way to do this
	valuesFiles := make([]string, 0)
	if helmArgs := p.GetHelmArgs(); helmArgs != "" {
		fields := strings.Fields(helmArgs)
		for idx, arg := range fields {
			if strings.HasPrefix(arg, "--values=") {
				f := strings.Join(strings.Split(arg, "=")[1:], "=")
				valuesFiles = append(valuesFiles, f)
				continue
			}
			if arg == "-f" || arg == "--values" {
				if len(fields) < idx {
					return nil, errors.New("got -f or --values helm flag without an argument")
				}
				valuesFiles = append(valuesFiles, fields[idx+1])
			}
		}
	}
	log.Debug("Combining the following values files for helm:", valuesFiles)
	var valuesContent string
	if len(valuesFiles) > 0 {
		for _, f := range valuesFiles {
			body, err := ioutil.ReadFile(f)
			if err != nil {
				return nil, err
			}
			valuesContent = valuesContent + string(body) + "\n"
		}
	}

	// package the chart to a temp file
	tmpDir, err := util.GetTempDir()
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)
	log.Debugf("Executing command: helm package %s -d %s", chartPath, tmpDir)
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
	packagedChart, err := ioutil.ReadFile(chartPkg)
	if err != nil {
		return nil, err
	}

	var out bytes.Buffer
	if err := helmCRTmpl.Execute(&out, map[string]string{
		"Name":          path.Base(chartPath),
		"ChartContent":  base64.StdEncoding.EncodeToString(packagedChart),
		"ValuesContent": valuesContent,
	}); err != nil {
		return nil, err
	}
	outBytes := out.Bytes()
	return &types.Artifact{
		Type: types.ArtifactManifest,
		Name: fmt.Sprintf("%s-helm-chart.yaml", path.Base(chartPath)),
		Body: ioutil.NopCloser(bytes.NewReader(outBytes)),
		Size: int64(len(outBytes)),
	}, nil
}
