#Shared/global workflows


Shared workflow provide predefined workflows:

- [Workflow using docker](workflow/docker) (build and services)
- [Cloud](workflow/cloud) (ec2, gce)
- [Testing](assert) (assert)
- [Tomcat](workflow)


Endly operates on local or remote resources referred as target in various service contracts. 
Target is of [Resource](https://github.com/viant/toolbox/blob/master/url/resource.go) type, 
defined as URL and credentials. 

To unify target naming the following function based methodology is used:

origin - version control origin
target - host resource where endly runs (usually 127.0.0.1 with localhost credentials)
buildTarget  - host resource where app is being built
serviceTarget - host resource where app service (i.e. datastore service) runs




