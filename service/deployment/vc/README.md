**Version Control Service**

This service uses SSH (exec) scraping to implement git/svn commands

| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- |
| version/control | status | run version control check on provided URL | [StatusRequest](serivce_contract.go) | [Info](serivce_contract.go)  |
| version/control | checkout | if target directory already  exist with matching origin URL, this action only pulls the latest changes without overriding local ones, otherwise full checkout | [CheckoutRequest](serivce_contract.go) | [Info](serivce_contract.go)   |
| version/control | commit | commit commits local changes to the version control | [CommitRequest](serivce_contract.go) | [Info](serivce_contract.go)   |
| version/control | pull | retrieve the latest changes from the origin | [PullRequest](serivce_contract.go) | [Info](serivce_contract.go)   |
