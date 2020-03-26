package atlasmap

//go:generate go run ./.packr/packr.go

import (
	"context"
	"github.com/RHsyseng/operator-utils/pkg/utils/openshift"
	"github.com/atlasmap/atlasmap-operator/pkg/apis/atlasmap/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/gobuffalo/packr/v2"
)

func (r *ReconcileAtlasMap) createConsoleYAMLSamples() {
	log.Info("Loading CR YAML samples.")
	box := packr.New("cryamlsamples", "../../../deploy/crs")
	if box.List() == nil {
		log.Error(nil, "CR YAML folder is empty. It is not loaded.")
		return
	}

	resMap := make(map[string]string)
	for _, filename := range box.List() {
		yamlStr, err := box.FindString(filename)
		if err != nil {
			resMap[filename] = err.Error()
			continue
		}
		apicurito := v1alpha1.AtlasMap{}
		err = yaml.Unmarshal([]byte(yamlStr), &apicurito)
		if err != nil {
			resMap[filename] = err.Error()
			continue
		}
		yamlSample, err := openshift.GetConsoleYAMLSample(&apicurito)
		if err != nil {
			resMap[filename] = err.Error()
			continue
		}
		err = r.client.Create(context.TODO(), yamlSample)
		if err != nil {
			resMap[filename] = err.Error()
			continue
		}
		resMap[filename] = "Applied"
	}

	for k, v := range resMap {
		log.Info("yaml ", " name: ", k, " status: ", v)
	}
}
