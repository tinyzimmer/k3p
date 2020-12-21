package images

import (
	"text/template"

	"github.com/Masterminds/sprig"
)

var registriesYamlTmpl = template.Must(template.New("").Funcs(sprig.TxtFuncMap()).Parse(`
mirrors:
  registry.private:
    endpoint:
      - https://localhost:30100

configs:
  "localhost:30100":
    auth:
      username: {{ .Username }}
      password: {{ .Password }}
    tls:
      ca_file: /etc/rancher/k3s/registry-ca.crt
`))

var registryTmpl = template.Must(template.New("").Funcs(sprig.TxtFuncMap()).Parse(`---
apiVersion: v1
kind: Secret
metadata:
  name: registry-tls
  namespace: kube-system
  labels:
    k8s-app: private-registry
type: kubernetes.io/tls
data:
  tls.crt: {{ .TLSCertificate | b64enc }}
  tls.key: {{ .TLSPrivateKey | b64enc }}
  ca.crt: {{ .TLSCACertificate | b64enc }}
---
apiVersion: v1
kind: Secret
metadata:
  name: kubenab-tls
  namespace: kube-system
  labels:
    k8s-app: kubenab
type: kubernetes.io/tls
data:
  tls.crt: {{ .TLSCertificate | b64enc }}
  tls.key: {{ .TLSPrivateKey | b64enc }}
  ca.crt: {{ .TLSCACertificate | b64enc }}
---
apiVersion: v1
kind: Secret
metadata:
  name: registry-htpasswd
  namespace: kube-system
  labels:
    k8s-app: private-registry
type: Opaque
data:
  htpasswd: {{ .RegistryAuthHtpasswd | b64enc }}
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: private-registry
  namespace: kube-system
  labels:
    k8s-app: private-registry
spec:
  replicas: 1
  selector:
    matchLabels:
      k8s-app: private-registry
  template:
    metadata:
      labels:
        k8s-app: private-registry
    spec:
      priorityClassName: system-cluster-critical
      volumes:
        - name: registry-data
          emptyDir: {}
        - name: registry-tls
          secret:
              secretName: registry-tls
        - name: registry-htpasswd
          secret:
              secretName: registry-htpasswd
      initContainers:
        - name: data-extractor
          image: private-registry-data:latest
          imagePullPolicy: Never
          command: ['tar', '-xvz', '--file=/var/registry-data.tgz', '--directory=/var/lib/registry']
          volumeMounts:
            - name: registry-data
              mountPath: /var/lib/registry
      containers:
        - name: private-registry
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
            - name: registry-tls
              mountPath: /etc/tls/certs
              readOnly: true
            - name: registry-htpasswd
              mountPath: /etc/auth
              readOnly: true
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kubenab
  namespace: kube-system
  labels:
    k8s-app: kubenab
spec:
  selector:
    matchLabels:
      k8s-app: kubenab
  replicas: 1
  template:
    metadata:
      labels:
        k8s-app: kubenab
    spec:
      priorityClassName: system-cluster-critical
      containers:
      - name: kubenab
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
        - name: tls
          mountPath: /etc/admission-controller/tls
      volumes:
        - name: tls
          secret:
            secretName: kubenab-tls
---
apiVersion: v1
kind: Service
metadata:
  name: private-registry
  namespace: kube-system
  labels:
    k8s-app: private-registry
spec:
  type: NodePort
  selector:
    k8s-app: private-registry
  ports:
    - port: 5000
      protocol: TCP
      targetPort: 5000
      nodePort: 30100
---
apiVersion: v1
kind: Service
metadata:
  name: kubenab
  namespace: kube-system
  labels:
    k8s-app: kubenab
spec:
  selector:
    k8s-app: kubenab
  type: ClusterIP
  ports:
  - port: 443
    protocol: "TCP"
    name: https
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
