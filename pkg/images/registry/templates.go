package registry

import (
	"bytes"
	"text/template"

	"github.com/Masterminds/sprig"
)

func executeTemplate(tmpl *template.Template, vars map[string]interface{}) ([]byte, error) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

var registryServicesTmpl = template.Must(template.New("").Funcs(sprig.TxtFuncMap()).Parse(`---
apiVersion: v1
kind: Service
metadata:
  name: {{ .RegistryK8sAppName }}
  namespace: {{ .RegistryNamespace }}
  labels:
    k8s-app: {{ .RegistryK8sAppName }}
spec:
  type: NodePort
  selector:
    k8s-app: {{ .RegistryK8sAppName }}
  ports:
    - port: 5000
      protocol: TCP
      targetPort: 5000
      nodePort: {{ .RegistryNodePort }}
---
apiVersion: v1
kind: Service
metadata:
  name: {{ .KubenabK8sAppName }}
  namespace: {{ .RegistryNamespace }}
  labels:
    k8s-app: {{ .KubenabK8sAppName }}
spec:
  selector:
    k8s-app: {{ .KubenabK8sAppName }}
  type: ClusterIP
  ports:
  - port: 443
    protocol: "TCP"
    name: https
`))

var registriesYamlTmpl = template.Must(template.New("").Funcs(sprig.TxtFuncMap()).Parse(`
mirrors:
  registry.private:
    endpoint:
      - https://localhost:{{ .RegistryNodePort }}

configs:
  "localhost:{{ .RegistryNodePort }}":
    auth:
      username: {{ .Username }}
      password: {{ .Password }}
    tls:
      ca_file: {{ .RegistryCAPath }}
`))

var registryAuthSecretTmpl = template.Must(template.New("").Funcs(sprig.TxtFuncMap()).Parse(`
apiVersion: v1
kind: Secret
metadata:
  name: {{ .RegistryAuthSecret }}
  namespace: {{ .RegistryNamespace }}
  labels:
    k8s-app: {{ .RegistryK8sAppName }}
type: Opaque
data:
  htpasswd: {{ .RegistryAuthHtpasswd | b64enc }}
`))

var registryTLSTmpl = template.Must(template.New("").Funcs(sprig.TxtFuncMap()).Parse(`---
apiVersion: v1
kind: Secret
metadata:
  name: {{ .RegistryTLSSecret }}
  namespace: {{ .RegistryNamespace }}
  labels:
    k8s-app: {{ .RegistryK8sAppName }}
type: kubernetes.io/tls
data:
  tls.crt: {{ .TLSCertificate | b64enc }}
  tls.key: {{ .TLSPrivateKey | b64enc }}
  ca.crt: {{ .TLSCACertificate | b64enc }}
---
apiVersion: v1
kind: Secret
metadata:
  name: {{ .KubenabTLSSecret }}
  namespace: {{ .RegistryNamespace }}
  labels:
    k8s-app: {{ .KubenabK8sAppName }}
type: kubernetes.io/tls
data:
  tls.crt: {{ .TLSCertificate | b64enc }}
  tls.key: {{ .TLSPrivateKey | b64enc }}
  ca.crt: {{ .TLSCACertificate | b64enc }}
---
apiVersion: admissionregistration.k8s.io/v1beta1
kind: MutatingWebhookConfiguration
metadata:
  name: kubenab-mutate
webhooks:
- name: kubenab-mutate.kubenab.com
  objectSelector:
    matchExpressions:
    - key: k8s-app
      operator: NotIn
      values: ["kube-dns", "kubenab", "private-registry", "metrics-server"]
  rules:
  - operations: [ "CREATE", "UPDATE" ]
    apiGroups: [""]
    apiVersions: ["v1"]
    resources: ["pods"]
  failurePolicy: Fail
  clientConfig:
    service:
      name: kubenab
      namespace: kube-system
      path: "/mutate"
    caBundle: {{ .TLSCACertificate | b64enc }}
`))

var registryDeploymentsTmpl = template.Must(template.New("").Funcs(sprig.TxtFuncMap()).Parse(`---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .RegistryK8sAppName }}
  namespace: {{ .RegistryNamespace }}
  labels:
    k8s-app: {{ .RegistryK8sAppName }}
spec:
  replicas: 1
  selector:
    matchLabels:
      k8s-app: {{ .RegistryK8sAppName }}
  template:
    metadata:
      labels:
        k8s-app: {{ .RegistryK8sAppName }}
    spec:
      priorityClassName: system-cluster-critical
      volumes:
        - name: registry-data
          emptyDir: {}
        - name: {{ .RegistryTLSSecret }}
          secret:
              secretName: {{ .RegistryTLSSecret }}
        - name: {{ .RegistryAuthSecret }}
          secret:
              secretName: {{ .RegistryAuthSecret }}
      initContainers:
        - name: data-extractor
          image: {{ .RegistryDataImage }}
          imagePullPolicy: Never
          command: ['tar', '-xvz', '--file=/var/registry-data.tgz', '--directory=/var/lib/registry']
          volumeMounts:
            - name: registry-data
              mountPath: /var/lib/registry
      containers:
        - name: {{ .RegistryK8sAppName }}
          image: registry:2
          imagePullPolicy: Never
          env:
            - name: REGISTRY_HTTP_TLS_CERTIFICATE
              value: /etc/tls/certs/tls.crt
            - name: REGISTRY_HTTP_TLS_KEY
              value: /etc/tls/certs/tls.key
            - name: REGISTRY_AUTH
              value: htpasswd
            - name: REGISTRY_AUTH_HTPASSWD_REALM
              value: "Private Registry Realm"
            - name: REGISTRY_AUTH_HTPASSWD_PATH
              value: /etc/auth/htpasswd
          ports:
            - containerPort: 5000
          volumeMounts:
            - name: registry-data
              mountPath: /var/lib/registry
            - name: {{ .RegistryTLSSecret }}
              mountPath: /etc/tls/certs
              readOnly: true
            - name: {{ .RegistryAuthSecret }}
              mountPath: /etc/auth
              readOnly: true
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .KubenabK8sAppName }}
  namespace: {{ .RegistryNamespace }}
  labels:
    k8s-app: {{ .KubenabK8sAppName }}
spec:
  selector:
    matchLabels:
      k8s-app: {{ .KubenabK8sAppName }}
  replicas: 1
  template:
    metadata:
      labels:
        k8s-app: {{ .KubenabK8sAppName }}
    spec:
      priorityClassName: system-cluster-critical
      containers:
      - name: {{ .KubenabK8sAppName }}
        image: {{ .KubenabImage }}
        imagePullPolicy: Never
        env:
          - name: DOCKER_REGISTRY_URL
            value: "registry.private"
          - name: WHITELIST_NAMESPACES
            value: "kube-system"
          - name: WHITELIST_REGISTRIES
            value: "registry.private,rancher"
          - name: REPLACE_REGISTRY_URL
            value: "true"
        ports:
          - containerPort: 443
            name: https
        volumeMounts:
        - name: {{ .KubenabTLSSecret }}
          mountPath: /etc/admission-controller/tls
      volumes:
        - name: {{ .KubenabTLSSecret }}
          secret:
            secretName: {{ .KubenabTLSSecret }}
`))
