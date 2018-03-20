### Assert



#### _Workflow run_
```bash
# call workflow with task start/stop
endly -w=assert actual=a expected=b

#or to test content of files 
endly -w=assert actual '@actual.json' expected '@expected.json'
```

#### _Pipeline run_

```bash
endly -p=run
```

@run.yaml
```yaml
pipeline:
  build:
    action: workflow:print
    message: building app ...
  test:
    workflow: assert
    actual:
      key1: value1
      key2: value20
      key3: value30
    expected:
      key1: value1
      key2: value2
      key3: value3
      key4: value4     
```




