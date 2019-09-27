## September 19 2019 0.43.0
    Integrated storage service with https://github.com/viant/afs
    Reanmed CopyWithCompression udf to GZipper - breaking change
    Renamed CopyWithCompressionAndCorruption udf to GZipContentCorrupter - breaking chagnge
    Added new storage operations:
       - storage:list
       - storage:exists
    
## September 19 2019 0.42.0
    * Added "aws/cloudwatchevents" service
    * Added  model.Extract.Required attribute
    * ForeignKey check switch patched for MySQL 
    * Added Schedule option to aws/lambda:deploy
    * Updated lambda various deployment docmentation
    
## September 11 2019 0.41.3
    * Patch, streamling sqs testing
    
## July 29 2019 0.41.2
    * Patched KMS GCP storage credentials relative naming
    * Change expect contract on msg:pull

## July 29 2019 0.41.1
    * Updated gcp:kms with policy binding
    * Added $gcp.serviceAccount variable for gcp:* services
    * Patched MySQL script parsing
    * Update KMS encrupt/decrypt to be compatible with gcloud
    * Update lambda attribute parsing
    
## July 21 2019 0.41.0
    * Removed process:stopAll - BREAKING CHANGE - please use process:stop
    * Extended process:stop to handle command match
    * Added Watch attribute to process:start
    * Added default target to process, sdk, deploy services
    * Streamined scp upload
    * Added logging flag to generic event
    * Patched exec secrets
    * Patched storage with multi asset and empty source
 
## July 10 2019 0.40.0
    * Added task.fail to control final endly workflow in case there was catch task
    * Added dsunit:checkSchema
    * Added exec:run AutoSudo mode
    * Patched .gcloudingnore filtering
    * Added baseLocation to deployment,sdk 
    * Remove config.xml transfer on tomcat deployment
    

## June 19 2019 0.39.2
    * Added @strictMapCheck@ validation directive (assertly)
    
## June 19 2019 0.39.1
    * Patch dsunit:expect CheckPolicy (dsunit)
        - BREAKING CHANGE, update dsunit.expect checkPolicy: 
            1 (SnapshotDatasetCheckPolicy) to get back to previous behaviour
              
    * Added exec.SessionID, and update terminal session access  
    
## June 12 2019 0.39.1
    * Update SSH session ID to use username@host:port
    * Added default ssh target for env.USER if ~/.secret/id_rsa or ~/.ssh/id_rsa is found
    * Patch ssh command output
    
## June 6 2019 0.39.0
    * Added cross vendor create table DDL with dsunit:dump
    * patched @queryFrom@ SQL builder 
 
## June 4 2019 0.38.0
    * Added command exit code check with CheckError attrbibute to SSH executor (exec)
    * Set CheckError on process:start
    * Embed exec.Options on process:start

## May 15 2019 0.37.4
    * Patched AsString conversion from []interface{} (viant/toolbox)
    * Added docker:deploy mem literal
    * Updated ubuntu endly docker build
    
## May 7 2019 0.37.0
    * Added VpcMatcher.Filter.SubnetExclusionTags (aws/ec2)
    * Minor patches
        
## April 21 2019 0.36.0

    * Added VpcMatcher to aws:lamba/deploy to match vpc by vpc or instance name
    * Added vc/git:commit
    * Minot patches
    
## April 14 2019 0.35.0
    * added gcp/kms service
    * added gcp/run service
    * added gcp/container service
    * updates gcp/* test
    * added generator function with $generator.next, $generator.prev, $generator.value, $generator.reset
    * updated sleep handling
    * updated async repeated action
    * minor pached
    
## March 18 2019 0.34.0
    * Added vc/git service to use go client, (original vc service uses SSH scraping)
    * Added aws/kms, aws/ssm services
        
## March 18 2019 0.33.0
    * Renamed ssh scraping based docker service to docker/ssh (breaking backward compatibility change)
    * Introduced docker service based on docker API client 
    * Updated/patched examples
    * Added dsunit.init - ping flag to wait for database getting online (dsunit.ping)
    * Updated template generator
    * Minor patches

## Feb 23 2019 0.32.2
    * Added dsunit:init ping option to wait for database get online
    * Updated endly usage endly [service:action | inlineworkflow.yaml]
    * Patched kubernetes:apply
    * Added kubernetes:forwardPorts
    * Updated kubernetes secrets documentation  
    * Added Public flag to cloud function deploy
    * Added Members field to cloud function deploy
    * Added Kubernetes auth plugin package

## Feb 23 2019 0.32.1
    * Added $Concat, $Merge udfs (toolbox/data)
    * Added available keys limit (assertly)

## Feb 18 2019 0.32.0
    * Added slack service
    * Added $AssetsToMap $BinaryAssetsToMap udfs (configmaps)
    * Renamed $gc. to $gcp. 
    * Patched inline yaml annotation error
    * Minor patches
    
## Feb 13 2019 0.31.0
    * Patched docker compose yaml port parsing
    * Patch json conversion
    * Added Select,Sum,Count,AsNumber udfs (toolbox/data/udf)
    * Patched Required mapping column name (dsunit)
    * Added <ds:empty> macro
    * Patched assertly NumericPresssion with zero value
    * Enhanced kubernetes service integraton (i.e. additional helper methods like: expose, run) 
    * RunWithoutLogging helper method
    
## Feb 5 2019 0.30.1
    * Patched ShowTable sql embedding issue (dsc)
    * Patched contract description (toolbox)
    * Added Elapsed,Sum,Count,AsNumber udf (toolbox)
    * Patched init database with recreate flag=true (dsunit)
    * Patched data comparision on time values with incomptible types (dsunit)
    * Aded kubernetes:get, kubernetes:create
    * Integrated more kubernetes resources

## Feb 1 2019 0.30.0
    * Renamed gc/* to gcp/* service ids - workflow breaking change
    * Added kubernetes/core and kubernetes/apps services
    * Added contract adapter code genrator util/adapter
    * Added BuildLowerCaseMapping utility

## Feb 1 2019 0.29.2
    * Pachted nil pointer on cloudfunctions:delete
    * Patched .gcloudignore functionality when deploy function
    
## Jan 31 2019 0.29.1
    * Pachted defer inline workflow task
    * Added gcc, remove gsunit from endly docker image
    * Added aws/iam policy variable substitution
    * Patched persiting rows with nil values (dsc)
    * Renamed package system/cloud/gc to system/cloud/gcp 
    
## Jan 30 2019 0.29.0
    * Added gc/cloudfuncton:deploy 
    * Added gc/cloudfuncton:call
    * Renamed aws/lamda:setupFunction to aws/lamda:deploy - this is breaking change.
    * Added RunInBackground helper method for long running action

## Jan 29 2019 0.28.0
    * Added direct cli level action call with command line parameters (-run=)
    * Patched nil pointer on trip runner/http:stressTest
    * Added endpoint:http file:// path support
    * Added workflow:setEnv action
    * Added docker:copy
    * Loading new line delimited JSON patched
    * Minor patches

## Jan 28 2019 0.27.1
    * Added -j endly cli option to list UDFs
    * Updated init attribute node without prefix  as action attribute
    * Patch storage.copy asset join with source and dest
    
## Jan 20 2019 0.27.0
    * Added gc/* services
    * Refactored and broke compatibility with former ec2/gce services
    * Moved system/docker to system/docker/ssh package    
    
## Jan 9 2019 0.26.0
    * Added endly Route OnRawRequest hook
    * Added aws/* services
    * Refactored aws/ec2 service
    * Added context level Loging for action
    * Minor patches
    
## Jan 7 2019 0.25.1
    * Resolved template variable collision with sub workflow templates
    * Patched non writable map data type mutation
    
## Jan 6 2019 0.25.0
    * Moved msg service to testing/msg package
    * Patched embeded non pointer struct conversion (toolbox)
    * Added firebase, casandra dependency (driver, template) 
    * Added dsunit.CompareRequest to check SQL based data from various db.
    * Added $context.secrets.${key} to expose credential config details like projectID, etc...
    * Add FollowRedirects option to http clinet (viant/toolbox)
    * Patched slice yaml variable conversion
    * Added message grouping
    * Added template tag failure level reporting for sub workflows
    * Stremlined CLI reporter
    
## December 28 2018 0.24.0
    * Added smtp endpoint
    * Moved event reporting msg package to model/msg
    * Moved messaging package to msg
    * Enhanced inline workflow node data substitution
    * Migrated e2e example to inline workflows and go 1.11 build
    * Ehnaced muli resource substitution with $argDataX and main yaml resource support
    * Added dataSource customization (response, response.Body) (http/runner:send)
    * Added workflow/default to fallback chain in external resource loading (inline workflow)
    * Patched processing unexported struct field (viant/toolbox)
    * Added QueryEscape udf (viant/toolbox)
    * BQ Streaming insert customization (viant/bgc)
    * Added Expect attribute in Query operation for optional valiation (SQL based validation) (viant/dsunit)   
    * Minor patches

## December 19 2018 0.23.1
    * Added logging option on abstract node level
    * Patched action tracking for cli execution path reporter  
    * Refactored yaml source kv paris into map for inline workflow action request attributes
    * Data substitution expression patches (toolbox)
    * Moved standard udf from neatly to toolbox/data/udf (neatly)
    * Added Expect attribute to http/runner Request type (data cohesion)
    * Minor patches

## December 12 2018 0.23.0
    * Added workflow scoped variable ($self.x)
    * Enhanced inline workflow task conversion process (init,post,when)
    * Added Values,Keys,IndexOf udf (neatly)
    * Enhanced multi parameters UDF call expression syntax (toolbox)
    * Renamed and moved pubsub service to messaging
    * Added AssertPath directive (assertly)

## December 5 2018 0.22.0
    * Refactored/streamlined expression parser
    * Added basic arithmetic support
    * Added workflow params and data to worklow state dedicated bucket
    * Enhanced criteria parser to work with UDF expression
    * Renamed ShareStateMode to ShareState on workflow:run request
    * Removed setting ShareState by inline workflow by default
    * Added elapsed time helper $elapsedToday.locale i.e. : ${elapsedToday.UTC}  
    * Added remianing time helper $remainingToday.locale i.e. : ${remainingToday.UTC}
    * Patched/refactored variable loading
        
## December 1 2018 0.21.2
    * Refactor asyn action to run with repeater like a regular action
    * Added keySensitive direction (assertly) 
    * Added CSVReder commond udf provider
    * Updated doc
    * Enabled diagnosctive with gops with -d switch

## November 28 2018 0.21.1
    * Patch nil pointer check in stress Test
    * Added coalesceWithZero directive, patched nil and numeric value validation (assertly)
    * Patched ToInt, ToFloat conversion to throw error if nil is supplied (toolbox)
    * Added Added LocationTimezone, TimeLayout attribute to FreezeRequest (dsunit)

## November 25 2018 0.21.0
    * Added http/runner:load action for HTTP endpoint stress testing
    * Added  NumericFloatPrecission
    * Added wildcard resource support for loading data section in actions template

## November 25 2018 0.20.0
    * Added  actions template support for inline workflow action template (neatly tag iterators)
    * Added multi asset supprot for inline workflow request (neatly like multi resource loading)
    * Added async flag to inline workflow at task level to allow parallel execution
    * Patched maching switch case with incompatible types
    * Patch assertly validator for nil expected time validation
    * Added regression format option to e2e project generator
    * Patched double execution of defer tasks
    * Updated documentation 
    

## November 18 2018 0.19.0
    * Added pubsub cloud messaging service
    * Added UDF service for registering udf with custom settings
    * Added generic protobuf UDF provider
    * Removed UDF Providers from service specific contracts in http/runner and storage services
    * Enhanced variable validation
    * Renamed Pipelines struct to InlineWorkflow
    * Updated documentation
    
## November 13 2018 0.18.0
    * Patched logger source.URL init to allow log validation with non schema based resources i.e. /tmp/logs/data as opposed to file:///tmp/logs/data
    * Updated logger validation documentation
    * Added $Len udf
    * Renamed ExpectedLogRecords in testing/log/service_contract.go to Expect for consistency (not backward compatible)
    * Added optional conversion from yaml kv paris to map in testing/log/service_contract.go  AssertRequest.Expect.Records

## November 8 2018 0.17.0
    * Added $dsconfig state keys with dsc.Config.params (i.e. $dsconfig.datasetId. $dsconfig.projectId)
    * Added dsunit.dump method to create schema DDL fro existing database
    * Refactor $timestamp. , $unix. to take advandate toolbox TimeAt method, i.e. ${unix.nowInUTC}, ${timestamp.5DaysAhead}
    * Added global $tzTime state function that uses time.RFC3339 time layout with toolbox.TimeAt semantic
    * Minor patches 

## October 30 2018 0.16.0
    * Added option to create setup or verification dataset with dsunit.freeze
    
## October 26 2018 0.15.1
    * Patched assertly numeric type casting criterion 
    * Renamed CopyHandlerUdf to Udf on storage copy request
    
## October 18 2018 0.15.0
    * Added SSH, Inline workflow runner option in e2e project generator 
    * Minor patches

## October 11 2018 0.14.1
    * Patched http trips cross reference expression substitution
    * Minor patches

## October 9 2018 0.14.0          
    * Added http recordng option -u
    * Added Avro UDFs
    * Added UDF providers for UDF registration on the fly
    * Minor patches

## October 2 2018 0.12.3          
    * Update logging with activity context (-d=true)
    * Minor patches
    
## July 12 2018 0.12.0          
    * Documents have been updated
    * Endly dependencies have been updated

## June 20 2018 0.11.1          
    * Patched record mapper nil pointer
    * Updated test generator links
    * Minor patches

## June 19 2018 0.11.0
    * Added endly -g option to generate a test project
    * Added log validator to project generator
    * Added documentation link to project generator options
    * Patched validation failure source matching
    * Patched task duplication when run with -t option
    * Renamed test project folder 'endly' to e2e in examples
    * Minor patches
    
## June 06 2018 0.10.1
  * Patched and enhanced storage service compression
  * Patched validation errors
  * Patched nil pointers on aws service
  * Minor patches  
  
## May 23 2018 0.10.0
    * Customized data setup in workflow generator
    * Added data validation option to workflow generator
    * Added repeater to rest send request
    * Assertly validation enhancement
    * Added actual datastore and expected dsunit validation data
    * Minor patches
    
    
    
#OLDER RELEASES    
    

## Sep 1 2017 (Alpha)

  * Initial Release.

## Jan 21 2018 0.1.0
    
   * First version release
    
## Feb 1  2018 0.2.0
    
    * Integrated with assertly data structure validation
    * Updated neatly to support @ for external resources, 
    * Added spaces (pipe has been already supported) for multi external resource separation 
    * Minor fixes
    
## Feb 12  2018 0.3.0
    * Updated udf expression to use $, instead of ! (no backward compatible change)
    * Simplified evaluation critiera
    * Refactored example workkflow with best practice
    * Maven build workflow optionally parameterized with custom .m2/settings.xml
    * Added request and response metadata with endly -s -a options
    * Added workflow task description with endly -t='?'
    
## Feb 18  2018 0.3.1
    * Updated service action request discovery (endly -s -a)
    * Streamlined error handling
    * Added tagIDs to WorkflowRunRequest option
    * Refactored docker container request
    * Refactored dsunit prepare/expect request
    * Updated documentation
    * Minor fixes
    
## Feb 23  2018 0.4.0
    * Integrated dsunit with assertly
    * Added unit/integrated test Run function
    * Patched tagIDs 
    * Moved secret credetnails file generator to endly -c option
    * Minor fixes
  
## March 4  2018 0.5.0
    * Updated criteria to support comprehensive conditional expression.
    * Reorganized services and dependencies
    * Minor fixes
    
## March 13  2018 0.6.0
    * Refactor and simpified exec.service
    * Added endly.Run helper
    * Renamed MatchStdout to When
    * Renamed MatchBody to When
    * Renamed RunCriteria to When
    * Renamed SkipCriteria to Skip
    * Renamed ExitCriteria to Exit
    * Renamed Credentials to Secrets
    * patched RepeatedReporter CLI event reporting
    
## March 29  2018 0.7.0
     * Refactor and simpified storage.service
     * Added When/Else to variable
     * Refactorored docker shared service added docs
     * Refactored SSH service with stdout listener (for instant stdout CLI reporting)
     * Minor patches
     * Renamed Credential to Credentials
     * Added more yaml examples
     * Add SSH testing utilities NewSSHRecodingContext, NewSSHReplayContext
     * Refactored and updated shared workflows        
     * Minor patched
     
## March 31 2018 0.7.1
     * Added Expec to all runner Run request and Assert field to response
     * Added automation with docker example
     * Reorganized documentation
     * Minor patches
     
## Apr 4 2018 0.7.3
     * Reverter order of adding os.path (to begining)
     * Added node sdk 
     * Minor patches
         
## Apr 6 2018 0.7.4
     * Minor patches
     
## Apr 13 2018 0.7.5
    * Add pipeline post
    * Patched neatly Cat udf
    * Added Pipeline state init block
    * Updated shared workflow
    * Added workflow generator
    
## Apr 16 2018 0.7.6
    * Patched workflow generator shared mem 
    * Merge docker compose pull request
    * Minor patches
    
## Apr 16 2018 0.7.7
    * Added map[interface{}]interface{} for non string key support
    * Minor patches

## Apr 17 2018 0.7.8
    * Patched CLI formatting
    * Updated workflow generator
    * Added when criteria to pipeline    
    * Minor patches

## Apr 18 2018 0.7.8
    * Expanded variable init expression/format
    
## Apr 18 2018 0.7.9
    * Added expect validation to storage Download
    * Enhanced expression parser for map key nested expression
    * Added catch, defer pipeline special tasks
    * Minor patches        

## Apr 24 2018 0.8.0
    * Merged pipeline into workflow
    * Added shrared state workflow mode
    * Update smtp with secret service
    * Enhanced workflow and inline workflow inspection with -p options
    * Patched gs remove folder
    * Added multi table mapping option to workflow generator
    * Minor patches
            

## Apr 27 2018 0.8.1
    * Patched multi keys validation with @indexBy@ directive
    * Added @sortText@ assertly directive
    * Refactored java maven build workflow
    * Minor patches
    
## May 02 2018 0.8.2
    * Added shared switchCase assertly validation key for shared data points
    * Patched workflow generator app with postgress issue
    * Enhnced @indexBy@ directive to use path exrp for nested sturcture on assertly
    * Minor patches
            
## May 07 2018 0.8.3
    * Added xunit summary report
    * Minor patches
            
## May 17 2018 0.9.0
    * Added explicit data attribute with  "@" prefix in inline workflows (pipeline)
    * Added multi datastore selection to workflow generator
    * Added autodiscovery to workflow generator
    * Update big query to support DDL schema file
    * Minor patches

