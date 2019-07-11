package util

import (
	"github.com/stretchr/testify/assert"
	"testing"
)





func Test_ShouldIgnoreLocation(t *testing.T) {
    var ignoreList =
    	[]string{
    	        "foa",
    	        "/bar",
			    "pre*",
			    "*suf",
			    "baz*qux",
			    "abc/",
				"**/cde",
			    "efk/**",
			    "go.mod",
			    "e2e/**",
                "deploy/**",
                "manager/**",


    	     }


	var useCases = []struct{
		description string
		ignoreList []string
		location string
		expect bool
	} {

		{
			ignoreList:ignoreList,
			description:"no rules apply, do not ignored",
			location:"yyy/nbn/kkk.txt",
			expect:false,
		},
		{
			ignoreList:ignoreList,
			description:"no rules apply, do not ignored",
			location:"kkk.txt",
			expect:false,
		},

		{
			ignoreList:ignoreList,
			description:"no rules apply, do not ignored",
			location:"aaa.b/ccc/ddd/eee/kkk.txt",
			expect:false,
		},
		{
			ignoreList:ignoreList,
			description:"ignored by rule foa",
			location:"manager/app/foa",
			expect:true,
		},
		{
			ignoreList:ignoreList,
			description:"ignored by rule foa",
			location:"manager/app/foa",
			expect:true,
		},
		{
			ignoreList:ignoreList,
			description:"ignored by rule foa",
			location:"foa",
			expect:true,
		},
		{
			ignoreList:ignoreList,
			description:"ignored by rule /bar",
			location:"bar/foo.text",
			expect:true,
		},
		{
			ignoreList:ignoreList,
			description:"do not ignored by rule /bar",
			location:"bar",
			expect:false,
		},

		{
			ignoreList:ignoreList,
			description:"ignored by rule pre*",
			location:"m/preaaa.text",
			expect:true,
		},
		{
			ignoreList:ignoreList,
			description:"ignored by rule pre*",
			location:"pre",
			expect:true,
		},



		{
			ignoreList:ignoreList,
			description:"ignored by rule *suf",
			location:"m/test.suf",
			expect:true,
		},
		{
			ignoreList:ignoreList,
			description:"ignored by rule *suf",
			location:"test2.suf",
			expect:true,
		},


		{
			ignoreList:ignoreList,
			description:"ignored by rule baz*qux",
			location:"m/bazaaaqux",
			expect:true,
		},
		{
			ignoreList:ignoreList,
			description:"ignored by rule baz*qux",
			location:"bazaaaqux",
			expect:true,
		},

		{
			ignoreList:ignoreList,
			description:"do not ignored by rule baz*qux",
			location:"bazaaaqux.test",
			expect:false,
		},

		{
			ignoreList:ignoreList,
			description:"do not ignored by rule baz*qux",
			location:"test.bazaaaqux",
			expect:false,
		},


		{
			ignoreList:ignoreList,
			description:"ignored by rule abc/",
			location:"abc/aaa.txt",
			expect:true,
		},

		{
			ignoreList:ignoreList,
			description:"ignored by rule abc/",
			location:"abc",
			expect:true,
		},


		{
			ignoreList:ignoreList,
			description:"do not ignored by rule **/cde",
			location:"a/cde/aaa.txt",
			expect:false,
		},

		{
			ignoreList:ignoreList,
			description:"ignored by rule **/cde",
			location:"a/cde",
			expect:true,
		},
		{
			ignoreList:ignoreList,
			description:"ignored by rule efk/**",
			location:"efk/aaa.txt",
			expect:true,
		},

		{
			ignoreList:ignoreList,
			description:"do not ignored by rule efk/**",
			location:"a/efk",
			expect:false,
		},
		{
			ignoreList:ignoreList,
			description:"ignored by rule go.mod",
			location:"vendor/gopkg.in/yaml.v2/go.mod",
			expect:true,
		},

		{
			ignoreList:ignoreList,
			description:"ignored by rule e2e/**",
			location:"e2e/app_cf.yaml",
			expect:true,
		},


		{
			ignoreList:ignoreList,
			description:"ignored by rule e2e/**",
			location:"e2e/vvv/app_cf.yaml",
			expect:true,
		},
	}




	for _, useCase := range useCases {
		actual := ShouldIgnoreLocation(useCase.location, useCase.ignoreList)
		assert.EqualValues(t, useCase.expect, actual, useCase.description)
	}
}