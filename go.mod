module github.com/viant/endly

go 1.17

require (
	cloud.google.com/go/container v1.3.1 // indirect
	cloud.google.com/go/firestore v1.3.0 // indirect
	cloud.google.com/go/kms v1.4.0 // indirect
	cloud.google.com/go/pubsub v1.3.1
	firebase.google.com/go v3.8.1+incompatible // indirect
	github.com/Azure/go-ansiterm v0.0.0-20170929234023-d6e3b3328b78 // indirect
	github.com/Azure/go-autorest/autorest v0.2.0 // indirect
	github.com/MichaelS11/go-cql-driver v0.1.1
	github.com/Microsoft/go-winio v0.4.12 // indirect
	github.com/adrianwit/dyndb v0.2.0
	github.com/adrianwit/fbc v0.1.1
	github.com/adrianwit/fsc v0.2.0
	github.com/adrianwit/mgc v0.2.0
	github.com/aerospike/aerospike-client-go v2.2.0+incompatible // indirect
	github.com/aws/aws-lambda-go v1.31.0
	github.com/aws/aws-sdk-go v1.44.12
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/docker/docker v0.7.3-0.20190515030239-f4b9142210e9
	github.com/docker/go-connections v0.4.0
	github.com/docker/go-units v0.4.0 // indirect
	github.com/docker/spdystream v0.0.0-20181023171402-6480d4af844c // indirect
	github.com/elazarl/goproxy v0.0.0-20190911111923-ecfe977594f1 // indirect
	github.com/emersion/go-smtp v0.11.1
	github.com/evanphx/json-patch v4.5.0+incompatible // indirect
	github.com/globalsign/mgo v0.0.0-20181015135952-eeefdecb41b8 // indirect
	github.com/go-errors/errors v1.4.2
	github.com/go-sql-driver/mysql v1.6.0
	github.com/gocql/gocql v0.0.0-20200815110948-5378c8f664e9 // indirect
	github.com/gomarkdown/markdown v0.0.0-20190222000725-ee6a7931a1e4 // indirect
	github.com/google/gofuzz v1.0.0 // indirect
	github.com/google/gops v0.3.6
	github.com/google/uuid v1.3.0
	github.com/googleapis/gnostic v0.3.0 // indirect
	github.com/gophercloud/gophercloud v0.2.0 // indirect
	github.com/gorilla/mux v1.7.3 // indirect
	github.com/gorilla/websocket v1.4.0 // indirect
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79 // indirect
	github.com/imdario/mergo v0.3.7 // indirect
	github.com/jhump/protoreflect v1.7.0
	github.com/json-iterator/go v1.1.6 // indirect
	github.com/klauspost/compress v1.7.1 // indirect
	github.com/klauspost/cpuid v1.2.1 // indirect
	github.com/klauspost/pgzip v1.2.1 // indirect
	github.com/lib/pq v1.10.5
	github.com/linkedin/goavro v2.1.0+incompatible
	github.com/logrusorgru/aurora v0.0.0-20190428105938-cea283e61946
	github.com/lunixbochs/vtclean v1.0.0
	github.com/lusis/slack-test v0.0.0-20190426140909-c40012f20018 // indirect
	github.com/mattbaird/jsonpatch v0.0.0-20171005235357-81af80346b1a
	github.com/mattn/go-sqlite3 v1.14.12
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/nlopes/slack v0.5.1-0.20190214144636-e73b432e20b0
	github.com/onsi/ginkgo v1.10.1 // indirect
	github.com/onsi/gomega v1.7.0 // indirect
	github.com/opencontainers/go-digest v1.0.0-rc1 // indirect
	github.com/opencontainers/image-spec v1.0.1 // indirect
	github.com/peterbourgon/diskv v2.0.1+incompatible // indirect
	github.com/pkg/errors v0.9.1
	github.com/satori/go.uuid v1.2.1-0.20181028125025-b2ce2384e17b
	github.com/segmentio/kafka-go v0.3.4
	github.com/sirupsen/logrus v1.4.2 // indirect
	github.com/spf13/pflag v1.0.3 // indirect
	github.com/stretchr/testify v1.7.1
	github.com/tebeka/selenium v0.9.3
	github.com/viant/afs v1.16.1-0.20220601210902-dc23d64dda15
	github.com/viant/afsc v1.8.1-0.20220525154204-272d99aaa19a
	github.com/viant/asc v0.5.0
	github.com/viant/assertly v0.9.1-0.20211210213130-9fc39dc0d8f0
	github.com/viant/bgc v0.8.0
	github.com/viant/dsc v0.16.2
	github.com/viant/dsunit v0.10.11-0.20221109235512-bdf35cb0327e
	github.com/viant/neatly v0.8.0
	github.com/viant/scy v0.3.2-0.20220818145333-129333b79ae7
	github.com/viant/toolbox v0.34.6-0.20220701174423-a46fd679bbc5
	github.com/yuin/gopher-lua v0.0.0-20190514113301-1cd887cd7036 // indirect
	golang.org/x/crypto v0.0.0-20220507011949-2cf3adece122
	golang.org/x/net v0.0.0-20220624214902-1bab6f366d9e
	golang.org/x/oauth2 v0.0.0-20220622183110-fd043fe589d2
	google.golang.org/api v0.90.0
	google.golang.org/protobuf v1.28.0
	gopkg.in/linkedin/goavro.v1 v1.0.5 // indirect
	gopkg.in/src-d/go-git.v4 v4.12.0
	gopkg.in/yaml.v2 v2.4.0
	gotest.tools v2.2.0+incompatible // indirect; indirec
	k8s.io/api v0.0.0-20190111032252-67edc246be36
	k8s.io/apimachinery v0.0.0-20190201131811-df262fa1a1ba
	k8s.io/client-go v10.0.0+incompatible
	k8s.io/klog v0.3.0 // indirect
	k8s.io/kube-openapi v0.0.0-20190401085232-94e1e7b7574c // indirect
	k8s.io/kubernetes v1.13.3
	sigs.k8s.io/yaml v1.1.0 // indirect
)

require github.com/golang-jwt/jwt/v4 v4.4.1

require (
	cloud.google.com/go v0.102.1 // indirect
	cloud.google.com/go/compute v1.7.0 // indirect
	cloud.google.com/go/iam v0.3.0 // indirect
	cloud.google.com/go/secretmanager v1.4.0 // indirect
	cloud.google.com/go/storage v1.22.1 // indirect
	contrib.go.opencensus.io/exporter/ocagent v0.4.12 // indirect
	github.com/Azure/go-autorest/autorest/adal v0.1.0 // indirect
	github.com/Azure/go-autorest/autorest/date v0.1.0 // indirect
	github.com/Azure/go-autorest/logger v0.1.0 // indirect
	github.com/Azure/go-autorest/tracing v0.1.0 // indirect
	github.com/census-instrumentation/opencensus-proto v0.2.1 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.0.0-20210816181553-5444fa50b93d // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible // indirect
	github.com/emersion/go-sasl v0.0.0-20161116183048-7e096a0a6197 // indirect
	github.com/emirpasic/gods v1.12.0 // indirect
	github.com/goccy/go-json v0.9.7 // indirect
	github.com/gogo/protobuf v1.2.0 // indirect
	github.com/golang/groupcache v0.0.0-20200121045136-8c9f03a8e57e // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/golang/snappy v0.0.3 // indirect
	github.com/google/btree v1.0.0 // indirect
	github.com/google/go-cmp v0.5.8 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.1.0 // indirect
	github.com/googleapis/gax-go/v2 v2.4.0 // indirect
	github.com/googleapis/go-type-adapters v1.0.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.16.0 // indirect
	github.com/hailocab/go-hostpool v0.0.0-20160125115350-e80d13ce29ed // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/kardianos/osext v0.0.0-20170510131534-ae77be60afb1 // indirect
	github.com/kevinburke/ssh_config v0.0.0-20180830205328-81db2a75821e // indirect
	github.com/konsorten/go-windows-terminal-sequences v1.0.1 // indirect
	github.com/lestrrat-go/backoff/v2 v2.0.8 // indirect
	github.com/lestrrat-go/blackmagic v1.0.0 // indirect
	github.com/lestrrat-go/httpcc v1.0.1 // indirect
	github.com/lestrrat-go/iter v1.0.1 // indirect
	github.com/lestrrat-go/jwx v1.2.25 // indirect
	github.com/lestrrat-go/option v1.0.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/pelletier/go-buffruneio v0.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/sergi/go-diff v1.0.0 // indirect
	github.com/src-d/gcfg v1.4.0 // indirect
	github.com/xanzy/ssh-agent v0.2.1 // indirect
	go.opencensus.io v0.23.0 // indirect
	golang.org/x/sync v0.0.0-20220601150217-0de741cfad7f // indirect
	golang.org/x/sys v0.0.0-20220624220833-87e55d714810 // indirect
	golang.org/x/term v0.0.0-20210927222741-03fcf44c2211 // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/time v0.0.0-20191024005414-555d28b269f0 // indirect
	golang.org/x/xerrors v0.0.0-20220609144429-65e65417b02f // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20220802133213-ce4fa296bf78 // indirect
	google.golang.org/grpc v1.48.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/src-d/go-billy.v4 v4.3.0 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776 // indirect
)
