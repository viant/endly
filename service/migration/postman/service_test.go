package postman

import (
	"strings"
	"testing"
)

const PostmanEnvironment = `
{
	"id": "eb212016-c2cc-4a44-ad7b-83eac2d2e4c2",
	"name": "ENVIRONMENT_1",
	"values": [
		{
			"key": "ENV1",
			"value": "ENVIRONMENT_1",
			"type": "default",
			"enabled": true
		},
		{
			"key": "SECRET1",
			"value": "SECRET_INITIAL",
			"type": "secret",
			"enabled": true
		},
		{
			"key": "host",
			"value": "localhost",
			"type": "default",
			"enabled": true
		},
		{
			"key": "port",
			"value": "8086",
			"type": "default",
			"enabled": true
		}
	],
	"_postman_variable_scope": "environment",
	"_postman_exported_at": "2023-05-08T01:48:02.151Z",
	"_postman_exported_using": "Postman/10.13.5"
}
`

const PostmanGlobals = `
{
	"id": "fce9da78-c5ba-4510-86f2-f2affed810ac",
	"values": [
		{
			"key": "GLOBAL_VAR1",
			"value": "2",
			"type": "default",
			"enabled": true
		}
	],
	"name": "Globals",
	"_postman_variable_scope": "globals",
	"_postman_exported_at": "2023-05-08T01:49:27.269Z",
	"_postman_exported_using": "Postman/10.13.5"
}
`

const PostmanRequest = `
{
	"info": {
		"_postman_id": "ce0b29d5-45c1-45bc-b3f3-eba3647cdc25",
		"name": "Endly Test",
		"schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json",
		"_exporter_id": "643429"
	},
	"item": [
		{
			"name": "Get Request",
			"request": {
				"method": "GET",
				"header": [
					{
						"key": "Test-Header-Secret",
						"value": "{{SECRET1}}",
						"type": "text"
					},
					{
						"key": "Test-Header-Global",
						"value": "{{GLOBAL_VAR1}}",
						"type": "text"
					}
				],
				"url": {
					"raw": "http://{{host}}:{{port}}?test=1",
					"protocol": "http",
					"host": [
						"{{host}}"
					],
					"port": "{{port}}",
					"query": [
						{
							"key": "test",
							"value": "1"
						}
					]
				}
			},
			"response": []
		},
		{
			"name": "Post Request",
			"request": {
				"method": "POST",
				"header": [
					{
						"key": "Test-Header-Secret",
						"value": "{{SECRET1}}",
						"type": "text"
					},
					{
						"key": "Test-Header-Global",
						"value": "{{GLOBAL_VAR1}}",
						"type": "text"
					},
					{
						"key": "Test-Header-Environment",
						"value": "{{ENV1}}",
						"type": "text"
					}
				],
				"body": {
					"mode": "raw",
					"raw": "{ \"TestBody\": \"Test Value\" }",
					"options": {
						"raw": {
							"language": "json"
						}
					}
				},
				"url": {
					"raw": "http://{{host}}:{{port}}",
					"protocol": "http",
					"host": [
						"{{host}}"
					],
					"port": "{{port}}"
				}
			},
			"response": []
		},
		{
			"name": "Put Request",
			"request": {
				"method": "PUT",
				"header": [
					{
						"key": "Test-Header-Secret",
						"value": "{{SECRET1}}",
						"type": "text"
					},
					{
						"key": "Test-Header-Global",
						"value": "{{GLOBAL_VAR1}}",
						"type": "text"
					},
					{
						"key": "Test-Header-Environment",
						"value": "{{ENV1}}",
						"type": "text"
					}
				],
				"body": {
					"mode": "raw",
					"raw": "{ \"TestBody\": \"Test Value\" }",
					"options": {
						"raw": {
							"language": "json"
						}
					}
				},
				"url": {
					"raw": "http://{{host}}:{{port}}",
					"protocol": "http",
					"host": [
						"{{host}}"
					],
					"port": "{{port}}"
				}
			},
			"response": []
		},
		{
			"name": "Patch Request",
			"request": {
				"method": "PATCH",
				"header": [
					{
						"key": "Test-Header-Secret",
						"value": "{{SECRET1}}",
						"type": "text"
					},
					{
						"key": "Test-Header-Global",
						"value": "{{GLOBAL_VAR1}}",
						"type": "text"
					},
					{
						"key": "Test-Header-Environment",
						"value": "{{ENV1}}",
						"type": "text"
					}
				],
				"body": {
					"mode": "raw",
					"raw": "{ \"TestBody\": \"Test Value\" }",
					"options": {
						"raw": {
							"language": "json"
						}
					}
				},
				"url": {
					"raw": "http://{{host}}:{{port}}",
					"protocol": "http",
					"host": [
						"{{host}}"
					],
					"port": "{{port}}"
				}
			},
			"response": []
		},
		{
			"name": "Delete Request",
			"request": {
				"method": "DELETE",
				"header": [
					{
						"key": "Test-Header-Secret",
						"value": "{{SECRET1}}",
						"type": "text"
					},
					{
						"key": "Test-Header-Global",
						"value": "{{GLOBAL_VAR1}}",
						"type": "text"
					},
					{
						"key": "Test-Header-Environment",
						"value": "{{ENV1}}",
						"type": "text"
					}
				],
				"body": {
					"mode": "raw",
					"raw": "{ \"TestBody\": \"Test Value\" }",
					"options": {
						"raw": {
							"language": "json"
						}
					}
				},
				"url": {
					"raw": "http://{{host}}:{{port}}",
					"protocol": "http",
					"host": [
						"{{host}}"
					],
					"port": "{{port}}"
				}
			},
			"response": []
		},
		{
			"name": "Copy Request",
			"protocolProfileBehavior": {
				"disableBodyPruning": true
			},
			"request": {
				"method": "COPY",
				"header": [
					{
						"key": "Test-Header-Secret",
						"value": "{{SECRET1}}",
						"type": "text"
					},
					{
						"key": "Test-Header-Global",
						"value": "{{GLOBAL_VAR1}}",
						"type": "text"
					},
					{
						"key": "Test-Header-Environment",
						"value": "{{ENV1}}",
						"type": "text"
					}
				],
				"body": {
					"mode": "raw",
					"raw": "{ \"TestBody\": \"Test Value\" }",
					"options": {
						"raw": {
							"language": "json"
						}
					}
				},
				"url": {
					"raw": "http://{{host}}:{{port}}",
					"protocol": "http",
					"host": [
						"{{host}}"
					],
					"port": "{{port}}"
				}
			},
			"response": []
		},
		{
			"name": "Head Request",
			"request": {
				"method": "HEAD",
				"header": [
					{
						"key": "Test-Header-Secret",
						"value": "{{SECRET1}}",
						"type": "text"
					},
					{
						"key": "Test-Header-Global",
						"value": "{{GLOBAL_VAR1}}",
						"type": "text"
					},
					{
						"key": "Test-Header-Environment",
						"value": "{{ENV1}}",
						"type": "text"
					}
				],
				"url": {
					"raw": "http://{{host}}:{{port}}",
					"protocol": "http",
					"host": [
						"{{host}}"
					],
					"port": "{{port}}"
				}
			},
			"response": []
		},
		{
			"name": "Options Request",
			"request": {
				"method": "OPTIONS",
				"header": [
					{
						"key": "Test-Header-Secret",
						"value": "{{SECRET1}}",
						"type": "text"
					},
					{
						"key": "Test-Header-Global",
						"value": "{{GLOBAL_VAR1}}",
						"type": "text"
					},
					{
						"key": "Test-Header-Environment",
						"value": "{{ENV1}}",
						"type": "text"
					}
				],
				"body": {
					"mode": "raw",
					"raw": "{ \"TestBody\": \"Test Value\" }",
					"options": {
						"raw": {
							"language": "json"
						}
					}
				},
				"url": {
					"raw": "http://{{host}}:{{port}}",
					"protocol": "http",
					"host": [
						"{{host}}"
					],
					"port": "{{port}}"
				}
			},
			"response": []
		},
		{
			"name": "Link Request",
			"request": {
				"method": "LINK",
				"header": [
					{
						"key": "Test-Header-Secret",
						"value": "{{SECRET1}}",
						"type": "text"
					},
					{
						"key": "Test-Header-Global",
						"value": "{{GLOBAL_VAR1}}",
						"type": "text"
					},
					{
						"key": "Test-Header-Environment",
						"value": "{{ENV1}}",
						"type": "text"
					}
				],
				"body": {
					"mode": "raw",
					"raw": "{ \"TestBody\": \"Test Value\" }",
					"options": {
						"raw": {
							"language": "json"
						}
					}
				},
				"url": {
					"raw": "http://{{host}}:{{port}}",
					"protocol": "http",
					"host": [
						"{{host}}"
					],
					"port": "{{port}}"
				}
			},
			"response": []
		},
		{
			"name": "Unlink Request",
			"request": {
				"method": "UNLINK",
				"header": [
					{
						"key": "Test-Header-Secret",
						"value": "{{SECRET1}}",
						"type": "text"
					},
					{
						"key": "Test-Header-Global",
						"value": "{{GLOBAL_VAR1}}",
						"type": "text"
					},
					{
						"key": "Test-Header-Environment",
						"value": "{{ENV1}}",
						"type": "text"
					}
				],
				"body": {
					"mode": "raw",
					"raw": "{ \"TestBody\": \"Test Value\" }",
					"options": {
						"raw": {
							"language": "json"
						}
					}
				},
				"url": {
					"raw": "http://{{host}}:{{port}}",
					"protocol": "http",
					"host": [
						"{{host}}"
					],
					"port": "{{port}}"
				}
			},
			"response": []
		},
		{
			"name": "Purge Request",
			"protocolProfileBehavior": {
				"disableBodyPruning": true
			},
			"request": {
				"method": "PURGE",
				"header": [
					{
						"key": "Test-Header-Secret",
						"value": "{{SECRET1}}",
						"type": "text"
					},
					{
						"key": "Test-Header-Global",
						"value": "{{GLOBAL_VAR1}}",
						"type": "text"
					},
					{
						"key": "Test-Header-Environment",
						"value": "{{ENV1}}",
						"type": "text"
					}
				],
				"body": {
					"mode": "raw",
					"raw": "{ \"TestBody\": \"Test Value\" }",
					"options": {
						"raw": {
							"language": "json"
						}
					}
				},
				"url": {
					"raw": "http://{{host}}:{{port}}",
					"protocol": "http",
					"host": [
						"{{host}}"
					],
					"port": "{{port}}"
				}
			},
			"response": []
		},
		{
			"name": "Lock Request",
			"request": {
				"method": "LOCK",
				"header": [
					{
						"key": "Test-Header-Secret",
						"value": "{{SECRET1}}",
						"type": "text"
					},
					{
						"key": "Test-Header-Global",
						"value": "{{GLOBAL_VAR1}}",
						"type": "text"
					},
					{
						"key": "Test-Header-Environment",
						"value": "{{ENV1}}",
						"type": "text"
					}
				],
				"body": {
					"mode": "raw",
					"raw": "{ \"TestBody\": \"Test Value\" }",
					"options": {
						"raw": {
							"language": "json"
						}
					}
				},
				"url": {
					"raw": "http://{{host}}:{{port}}",
					"protocol": "http",
					"host": [
						"{{host}}"
					],
					"port": "{{port}}"
				}
			},
			"response": []
		},
		{
			"name": "Unlock Request",
			"protocolProfileBehavior": {
				"disableBodyPruning": true
			},
			"request": {
				"method": "UNLOCK",
				"header": [
					{
						"key": "Test-Header-Secret",
						"value": "{{SECRET1}}",
						"type": "text"
					},
					{
						"key": "Test-Header-Global",
						"value": "{{GLOBAL_VAR1}}",
						"type": "text"
					},
					{
						"key": "Test-Header-Environment",
						"value": "{{ENV1}}",
						"type": "text"
					}
				],
				"body": {
					"mode": "raw",
					"raw": "{ \"TestBody\": \"Test Value\" }",
					"options": {
						"raw": {
							"language": "json"
						}
					}
				},
				"url": {
					"raw": "http://{{host}}:{{port}}",
					"protocol": "http",
					"host": [
						"{{host}}"
					],
					"port": "{{port}}"
				}
			},
			"response": []
		},
		{
			"name": "Propfind Request",
			"request": {
				"method": "PROPFIND",
				"header": [
					{
						"key": "Test-Header-Secret",
						"value": "{{SECRET1}}",
						"type": "text"
					},
					{
						"key": "Test-Header-Global",
						"value": "{{GLOBAL_VAR1}}",
						"type": "text"
					},
					{
						"key": "Test-Header-Environment",
						"value": "{{ENV1}}",
						"type": "text"
					}
				],
				"body": {
					"mode": "raw",
					"raw": "{ \"TestBody\": \"Test Value\" }",
					"options": {
						"raw": {
							"language": "json"
						}
					}
				},
				"url": {
					"raw": "http://{{host}}:{{port}}",
					"protocol": "http",
					"host": [
						"{{host}}"
					],
					"port": "{{port}}"
				}
			},
			"response": []
		},
		{
			"name": "View Request",
			"request": {
				"method": "VIEW",
				"header": [
					{
						"key": "Test-Header-Secret",
						"value": "{{SECRET1}}",
						"type": "text"
					},
					{
						"key": "Test-Header-Global",
						"value": "{{GLOBAL_VAR1}}",
						"type": "text"
					},
					{
						"key": "Test-Header-Environment",
						"value": "{{ENV1}}",
						"type": "text"
					}
				],
				"body": {
					"mode": "raw",
					"raw": "{ \"TestBody\": \"Test Value\" }",
					"options": {
						"raw": {
							"language": "json"
						}
					}
				},
				"url": {
					"raw": "http://{{host}}:{{port}}",
					"protocol": "http",
					"host": [
						"{{host}}"
					],
					"port": "{{port}}"
				}
			},
			"response": []
		}
	],
	"event": [
		{
			"listen": "prerequest",
			"script": {
				"type": "text/javascript",
				"exec": [
					""
				]
			}
		},
		{
			"listen": "test",
			"script": {
				"type": "text/javascript",
				"exec": [
					"pm.test(\"Status code is 200\", function () {",
					"  pm.response.to.have.status(200);",
					"});"
				]
			}
		}
	],
	"variable": [
		{
			"key": "var1",
			"value": "0",
			"type": "string"
		}
	]
}
`

// No _postman_id under info node
const PostmanBadRequest = `
{
	"info": {
		"name": "Endly Test",
		"schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json",
		"_exporter_id": "643429"
	},
	"item": [
		{
			"name": "Get Request",
			"request": {
				"method": "GET",
				"header": [
					{
						"key": "Test-Header-Secret",
						"value": "{{SECRET1}}",
						"type": "text"
					},
					{
						"key": "Test-Header-Global",
						"value": "{{GLOBAL_VAR1}}",
						"type": "text"
					}
				],
				"url": {
					"raw": "http://{{host}}:{{port}}?test=1",
					"protocol": "http",
					"host": [
						"{{host}}"
					],
					"port": "{{port}}",
					"query": [
						{
							"key": "test",
							"value": "1"
						}
					]
				}
			},
			"response": []
		}
	],
	"event": [
		{
			"listen": "prerequest",
			"script": {
				"type": "text/javascript",
				"exec": [
					""
				]
			}
		},
		{
			"listen": "test",
			"script": {
				"type": "text/javascript",
				"exec": [
					"pm.test(\"Status code is 200\", function () {",
					"  pm.response.to.have.status(200);",
					"});"
				]
			}
		}
	],
	"variable": [
		{
			"key": "var1",
			"value": "0",
			"type": "string"
		}
	]
}
`

const BadJson = `
{
	"bad":"comma",
}
`

const EmptyString = ""

const GoodJsonNotPostman = `
{
	"good":"json"
}
`

func TestNegativeParseTest(t *testing.T) {
	//Bad Json
	r1 := strings.NewReader(BadJson)

	_, err := parsePostmanReader(r1)

	if err == nil {
		t.Fatalf(`Parsing bad json did not throw error as expected`)
	}

	//Empty String
	r2 := strings.NewReader(EmptyString)

	_, err = parsePostmanReader(r2)

	if err == nil {
		t.Fatalf(`Parsing empty string did not throw error as expected`)
	}

	//Good Json Not Postman
	r3 := strings.NewReader(GoodJsonNotPostman)

	o, err := parsePostmanReader(r3)

	if err != nil {
		t.Fatalf(`Parsing good json so should not throw errors`)
	}

	if o.nodes != nil && o.objectType != notPostman {
		t.Fatalf(`This is not postman json so result should be nil`)
	}

	//Good Json bad Postman Request
	r4 := strings.NewReader(PostmanBadRequest)

	o, err = parsePostmanReader(r4)

	if err != nil {
		t.Fatalf(`Parsing good json so should not throw errors`)
	}

	if o.nodes != nil && o.objectType != notPostman {
		t.Fatalf(`This is a bad postman request so result should be nil`)
	}
}

func TestParseEnvironment(t *testing.T) {
	r1 := strings.NewReader(PostmanEnvironment)

	o, err := parsePostmanReader(r1)
	if err != nil {
		t.Fatalf(`Parsing failed due to: %v`, err)
	}

	if o.objectType != environment {
		t.Fatalf(`expected object type %v but got %v`, environment, o.objectType)
	}
}

func TestParseGlobals(t *testing.T) {
	r1 := strings.NewReader(PostmanGlobals)

	o, err := parsePostmanReader(r1)
	if err != nil {
		t.Fatalf(`Parsing failed due to: %v`, err)
	}

	if o.objectType != globals {
		t.Fatalf(`expected object type %v but got %v`, globals, o.objectType)
	}
}

func TestParseRequest(t *testing.T) {
	r1 := strings.NewReader(PostmanRequest)

	o, err := parsePostmanReader(r1)
	if err != nil {
		t.Fatalf(`Parsing failed due to: %v`, err)
	}

	if o.objectType != requests {
		t.Fatalf(`expected object type %v but got %v`, requests, o.objectType)
	}
}

func TestRunBuilder(t *testing.T) {
	b := NewRunBuilder()
	b.addVariable("test", "1")
	r := b.addRequest("Get", "http://localhost:8086", "Get Request")
	r.Body = "{\"test\": \"value\"}"
	r.addHeader("Content-Type", []string{"application/json"})
	c := r.addCookie()
	c.Domain = "localhost"
	c.Expires = "Tue, 07 May 2024 01:19:28 GMT"
	c.HttpOnly = true
	c.MaxAge = 1
	c.Name = "Cookie_1"
	c.Raw = "Cookie_1=value; Path=/; Expires=Tue, 07 May 2024 01:19:28 GMT;"
	c.RawExpires = "Expires=Tue, 07 May 2024 01:19:28 GMT"
	c.SameSite = 1
	c.Secure = false
	c.Unparsed = []string{"Cookie_1=value", "Path=/", "Expires=Tue, 07 May 2024 01:19:28 GMT"}
	c.Path = "/"
	c.Value = "1"
	e := b.addEnvironment("Environment 1")
	e.addVariable("host", "localhost")
	e.addVariable("port", "8081")
	e = b.addEnvironment("Environment 2")
	e.addVariable("host", "localhost")
	e.addVariable("port", "8082")

	if !(len(b.requests) == 1 && len(b.requests[0].Cookies) == 1 &&
		b.requests[0].Header["Content-Type"][0] == "application/json") &&
		b.variables["test"] == "1" && len(b.environments) == 2 && b.environments[1].Variables["port"] == "8082" {
		t.Fatalf("runBuilder is broken")
	}

	_, err := b.requests[0].TOJson()
	if err != nil {
		t.Fatalf(`TOJson failed with the following: %q`, err)
	}

	_, err = b.environments[0].TOJson()
	if err != nil {
		t.Fatalf(`TOJson failed with the following: %q`, err)
	}
}

func TestConvertToBuilder(t *testing.T) {
	r1 := strings.NewReader(PostmanEnvironment)

	e, err := parsePostmanReader(r1)
	if err != nil {
		t.Fatalf(`Parsing failed due to: %v`, err)
	}

	r1 = strings.NewReader(PostmanGlobals)

	g, err := parsePostmanReader(r1)
	if err != nil {
		t.Fatalf(`Parsing failed due to: %v`, err)
	}

	r1 = strings.NewReader(PostmanRequest)

	r, err := parsePostmanReader(r1)
	if err != nil {
		t.Fatalf(`Parsing failed due to: %v`, err)
	}

	b := convertToRunBuilder([]*postmanObject{e, g, r})

	if !(b.environments[0].Variables["ENV1"] == "ENVIRONMENT_1" &&
		b.variables["GLOBAL_VAR1"] == "2" &&
		len(b.requests) == 15) {
		t.Fatalf("Conversion to RunBuilder failed")
	}
}

func TestMakeDirOrFileName(t *testing.T) {
	s := makeDirOrFileName("!This #$%^is a_test")
	if s != "This_is_a_test" {
		t.Fatalf("Making a safe Directory or File name failed")
	}
}
