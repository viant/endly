# Shared/global workflow 

## TODO: deprecate 

Threat this functionality as deprecated, inline workflow are simply enough thus no need for shared/global workflows.



Shared workflow provide predefined workflows (old neatly format):

- [Services](workflow/service) (datastore/caching services)
- [App](workflow/app) (build/deployment,publishing including docker)



Endly operates on local or remote resources referred as target in various service contracts. 
Target is of [Resource](https://github.com/viant/toolbox/blob/master/url/resource.go) type, 
defined as URL and credentials. 

To unify target naming the following function based methodology is used:

- origin - version control origin
- target - host resource where endly runs (usually 127.0.0.1 with localhost credentials)
- buildTarget  - host resource where app is being built
- appTarget - host whre app is deployed and runs
- serviceTarget - host resource where app service (i.e. datastore service) runs


_Note_
All shared workflow/resources are compiled into target endly binary, make sure that you run ./gen.go each time any resource under shared folder has been modified.

