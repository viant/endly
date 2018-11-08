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




    



    
