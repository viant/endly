package cloudwatchevents

// DefaultRolePolicy represents defaul role policy
const DefaultRolePolicy = `{
   "Version": "2012-10-17",
   "Statement": [
      {
         "Sid": "CloudWatchEventsFullAccess",
         "Effect": "Allow",
         "Action": "events:*",
         "Resource": "*"
      },
      {
         "Sid": "IAMPassRoleForCloudWatchEvents",
         "Effect": "Allow",
         "Action": "iam:PassRole",
         "Resource": "arn:aws:iam::*:role/AWS_Events_Invoke_Targets"
      }      
   ]
}`

// DefaultTrustRelationship represents default trust relationship
const DefaultTrustRelationship = `{
   "Version": "2012-10-17",
   "Statement": [
      {
         "Effect": "Allow",
         "Principal": {
            "Service": [
				"events.amazonaws.com",
				"lambda.amazonaws.com"
			]

         },
         "Action": "sts:AssumeRole"
      }      
   ]
}`

//appRequestSubmitted
