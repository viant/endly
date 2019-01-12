package iam

import 	"github.com/aws/aws-sdk-go/service/iam"

//RecreateRoleInput drops role if exist to create a new one
type RecreateRoleInput iam.CreateRoleInput

//DropRoleInput deataches all role polices and deletes role
type DropRoleInput iam.DeleteRoleInput



