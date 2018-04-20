
## Sep 1 2017 (Alpha)

  * Initial Release.

## Jan 21 2018 0.1.0
    
   * First version release
    
## Feb 1  2018 0.2.0
    
    * Integrated with assertly data structure validation
    * Update neatly to support @ for external resources, 
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
    * Minor patches        
    