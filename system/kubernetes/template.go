package core

const podSpec = `
serviceAccountName: $ServiceAccount
dNSPolicy: $DNSPolicy
restartPolicy: $RestartPolicy
containers:
  - name: $Name
    image: $Image
    imagePullPolicy: $ImagePullPolicy
    env: $Envs
    args: $Args
    resources: $Resources
    commands: $Commands
    ports: $Ports
`

var runTemplates = map[string]string{
	DeploymentAppsV1GeneratorName: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: $Name
  labels: $Labels
spec:
  replicas: $Replicas
  selector: $LabelSelector
  strategy:
    rollingUpdate:
      maxSurge:
        IntVal: 1
        Type: 0
      maxUnavailable:
        IntVal: 1
        Type: 0
    type: RollingUpdate
  template:
    metadata:
      labels: $Labels
    spec: $Spec
`,

	RunV1GeneratorName: `apiVersion: v1
kind: ReplicationController
metadata:
  name: $Name
  labels: $Labels
spec:
  replicas: $Replicas
  selector: $Labels
  template:
    metadata:
      labels: $Labels
    spec: $Spec
`,
	RunPodV1GeneratorName: `apiVersion: v1
kind: Pod
metadata:
  name: $Name
  labels: $Labels
spec: $Spec`,
	JobV1GeneratorName: `apiVersion: batch/v1
kind: Job
metadata:
  name: $Name
  labels: $Labels
spec:
  template:
    metadata:
      labels: $Labels
    spec: $Spec`,
	CronJobV1Beta1GeneratorName: `apiVersion: batch/v1beta1
kind: CronJob
metadata:
  name: $Name
  labels: $Labels
spec:
  schedule: $Schedule
  concurrencyPolicy: AllowConcurrent
  selector: $Labels
  jobTemplate:
    metadata:
      labels: $Labels
    spec: $Spec
`,
}

var exposeTemplates = map[string]string{
	ServiceV1GeneratorName: `apiVersion: v1
kind: Service
metadata:
  name: $Name
  labels: $Labels
spec:
  selector: $Selector
  ports: $Ports
  clusterIP: $ClusterIP
  type: $Type
  externalIPs: $ExternalIPs
  sessionAffinity: $SessionAffinity
  loadBalancerIP: $LoadBalancerIP
  externalName: $ExternalName
  healthCheckNodePort: $HealthCheckNodePort
  sessionAffinityConfig: $SessionAffinityConfig
`,
}
