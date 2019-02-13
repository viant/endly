package core

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/viant/assertly"
	"github.com/viant/toolbox"
	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestGenerateRequest(t *testing.T) {

	useCases := []struct {
		descritption string
		runRequest   *RunRequest
		expect       string
		expectMeta   *v1.TypeMeta
		hasError     bool
	}{{

		descritption: "single instance generator",
		runRequest: &RunRequest{
			Name:  "nginx",
			Image: "nginx",
		},
		expect: `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    run: nginx
  name: nginx
spec:
  selector:
    matchLabels:
      run: nginx
  template:
    metadata:
      labels:
        run: nginx
    spec:
      containers:
      - image: nginx
        imagePullPolicy: Always
        name: nginx
      restartPolicy: Always`,
	},
		{
			descritption: "single instance with env, with exposed port",
			runRequest: &RunRequest{
				Name:  "hazelcast",
				Image: "hazelcast",
				Port:  5701,
				Env: map[string]string{
					"DNS_DOMAIN":    "cluster",
					"POD_NAMESPACE": "default",
				},
			},
			expect: `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    run: hazelcast
  name: hazelcast
spec:
  replicas: 1
  selector:
    matchLabels:
      run: hazelcast
  strategy:
    rollingUpdate:
      maxSurge:
        IntVal: 1
      maxUnavailable:
        IntVal: 1
    type: RollingUpdate
  template:
    metadata:
      labels:
        run: hazelcast
    spec:
      containers:
      - env:
        - name: DNS_DOMAIN
          value: cluster
        - name: POD_NAMESPACE
          value: default
        image: hazelcast
        imagePullPolicy: Always
        name: hazelcast
        ports:
        - containerPort: 5701
          hostPort: 0
      dNSPolicy: ClusterFirst
      restartPolicy: Always`,
		},
		{
			descritption: " replicated instance of nginx.",
			runRequest: &RunRequest{
				Name:     "nginx",
				Image:    "nginx",
				Replicas: 5,
			},
			expect: `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    run: nginx
  name: nginx
spec:
  replicas: 5
  selector:
    matchLabels:
      run: nginx
  strategy:
    rollingUpdate:
      maxSurge:
        IntVal: 1
      maxUnavailable:
        IntVal: 1
    type: RollingUpdate
  template:
    metadata:
      labels:
        run: nginx
    spec:
      containers:
      - image: nginx
        imagePullPolicy: Always
        name: nginx
      dNSPolicy: ClusterFirst
      restartPolicy: Always`,
		},

		{
			descritption: "single a pod",
			runRequest: &RunRequest{
				Name:          "nginx",
				Image:         "nginx",
				RestartPolicy: "Never",
			},
			expect: `apiVersion: v1
kind: Pod
metadata:
  labels:
    run: nginx
  name: nginx
spec:
  containers:
  - image: nginx
    imagePullPolicy: Always
    name: nginx
  dNSPolicy: ClusterFirst
  restartPolicy: Never`,
		},
		{
			descritption: "job template",
			runRequest: &RunRequest{
				Name:  "pi",
				Image: "perl",
				Commands: []string{
					"perl -Mbignum=bpi -wle 'print bpi(2000)",
				},
				RestartPolicy: "OnFailure",
			},
			expect: `apiVersion: batch/v1
kind: Job
metadata:
  labels:
    run: pi
  name: pi
spec:
  template:
    metadata:
      labels:
        run: pi
    spec:
      containers:
      - commands:
        - perl -Mbignum=bpi -wle 'print bpi(2000)
        image: perl
        imagePullPolicy: Always
        name: pi
      dNSPolicy: ClusterFirst
      restartPolicy: OnFailure
`,
		},
		{
			descritption: "scheduler job template",
			runRequest: &RunRequest{
				Name:  "pi",
				Image: "perl",
				Commands: []string{
					"perl -Mbignum=bpi -wle 'print bpi(2000)",
				},
				Schedule: "0/1 * * * ?",
			},
			expect: `apiVersion: batch/v1beta1
kind: CronJob
metadata:
  labels:
    run: pi
  name: pi
spec:
  concurrencyPolicy: AllowConcurrent
  jobTemplate:
    metadata:
      labels:
        run: pi
    spec:
      containers:
      - commands:
        - perl -Mbignum=bpi -wle 'print bpi(2000)
        image: perl
        imagePullPolicy: Always
        name: pi
      dNSPolicy: ClusterFirst
      restartPolicy: Always
  schedule: 0/1 * * * ?
  selector:
    run: pi
`,
		},
	}

	for _, useCase := range useCases {
		_ = useCase.runRequest.Init()
		params, err := NewRunTemplateParams(useCase.runRequest)
		assert.Nil(t, err, useCase.descritption)

		var actualYAML string
		err = GenerateRequest(useCase.runRequest.Template, runTemplates, params, func(meta *ResourceMeta, data map[string]interface{}) error {
			data = toolbox.DeleteEmptyKeys(data)
			if YAML, err := yaml.Marshal(data); err == nil {
				actualYAML = string(YAML)
			}
			return nil
		})

		var actualMap = make(map[string]interface{})
		err = yaml.Unmarshal([]byte(actualYAML), &actualMap)
		assert.Nil(t, err)
		var expectMap = make(map[string]interface{})
		err = yaml.Unmarshal([]byte(useCase.expect), &expectMap)

		assert.Nil(t, err, useCase.descritption)
		if !assertly.AssertValues(t, expectMap, actualMap, useCase.descritption) {
			fmt.Printf("%v\n", actualYAML)
		}
	}
}
